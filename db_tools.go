package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/lemmego/api/db"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- the read-only guard ----------------------------------------------------

// readOnlyStatements are the statement heads a query tool accepts when writes
// are off.
//
// An allowlist, not a denylist of DELETE/UPDATE/DROP. A denylist has to
// enumerate every way SQL can change something — and it will miss one, because
// the list includes TRUNCATE, GRANT, COPY, VACUUM, ALTER, CREATE, COMMENT,
// SECURITY LABEL, REINDEX, CLUSTER, and anything a future server version adds.
// Getting that list wrong means an agent drops a table.
var readOnlyStatements = []string{"select", "with", "show", "explain", "table", "values"}

// errNotReadOnly explains the refusal, naming the fix, because an agent that
// is told only "refused" will try to phrase its way around it.
const errNotReadOnly = "this server is read-only: %q is not a read statement. " +
	"Only %s are accepted. Build the server with mcp.WithWrites() to allow others."

// checkReadOnly reports whether sql may run, and why not when it may not.
func checkReadOnly(sql string) error {
	trimmed := strings.TrimSpace(sql)
	// A trailing semicolon is fine; a second statement after it is not,
	// because the guard only ever inspects the first and `SELECT 1; DROP
	// TABLE users` would pass it.
	if inner := strings.TrimSuffix(trimmed, ";"); strings.Contains(inner, ";") {
		return fmt.Errorf("only one statement may be sent at a time, and this contains several")
	}

	// Leading parentheses first: `(SELECT 1)` and `((SELECT 1))` are ordinary
	// reads, and without this the head would be the whole statement and be
	// refused. Trimmed rather than counted, because the nesting depth does
	// not change which statement it is.
	head := strings.TrimLeft(trimmed, "( \t\n\r")
	if boundary := strings.IndexFunc(head, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '('
	}); boundary > 0 {
		head = head[:boundary]
	}
	head = strings.ToLower(head)

	for _, allowed := range readOnlyStatements {
		if head == allowed {
			return nil
		}
	}
	return fmt.Errorf(errNotReadOnly, head, strings.Join(readOnlyStatements, ", "))
}

// --- database-connections ---------------------------------------------------

type dbConnectionsOutput struct {
	Registered bool   `json:"registered"`
	Name       string `json:"name,omitempty"`
	Dialect    string `json:"dialect,omitempty"`
	Default    string `json:"configured_default,omitempty"`
	DSN        string `json:"dsn,omitempty"`
	OpenConns  int    `json:"open_connections,omitempty"`
	InUse      int    `json:"in_use,omitempty"`
}

func (s *Server) addDatabaseConnections(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name: "database-connections",
		Description: "Which database this application is actually connected to — the dialect, " +
			"the connection name, the DSN with its credentials withheld, and the live pool " +
			"statistics. Use it to confirm you are reasoning about the right database before " +
			"running a query or writing a migration.",
	}, s.databaseConnections)
}

func (s *Server) databaseConnections(_ context.Context, _ *sdk.CallToolRequest, _ emptyInput) (*sdk.CallToolResult, dbConnectionsOutput, error) {
	cfg := s.app.Config()
	out := dbConnectionsOutput{Default: asString(cfg.Get("sql.default"))}

	conn, ok := db.Resolve(s.app)
	if !ok || conn == nil {
		return text("no SQL connection is registered; this project may have been scaffolded without a database"), out, nil
	}

	out.Registered = true
	out.Name = conn.Name()
	out.Dialect = string(conn.Dialect())
	out.DSN = RedactDSN(asString(cfg.Get(fmt.Sprintf("sql.connections.%s.dsn", conn.Name()))))
	if pool := conn.SQLDB(); pool != nil {
		stats := pool.Stats()
		out.OpenConns = stats.OpenConnections
		out.InUse = stats.InUse
	}

	var b strings.Builder
	fmt.Fprintf(&b, "connection %q, dialect %s\n", out.Name, out.Dialect)
	if out.DSN != "" {
		fmt.Fprintf(&b, "dsn: %s\n", out.DSN)
	}
	fmt.Fprintf(&b, "pool: %d open, %d in use\n", out.OpenConns, out.InUse)
	return text(b.String()), out, nil
}

// --- database-schema --------------------------------------------------------

type dbSchemaInput struct {
	Table string `json:"table,omitempty" jsonschema:"one table to describe in full; omit to list every table"`
}

type columnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Default  string `json:"default,omitempty"`
}

type tableInfo struct {
	Name    string       `json:"name"`
	Columns []columnInfo `json:"columns,omitempty"`
}

type dbSchemaOutput struct {
	Dialect string      `json:"dialect"`
	Tables  []tableInfo `json:"tables"`
}

func (s *Server) addDatabaseSchema(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name: "database-schema",
		Description: "The live schema: every table, or one table's columns with their types, " +
			"nullability and defaults. This reads the database rather than the migration files, " +
			"so it shows what is actually there — including anything applied by hand or by a " +
			"migration that half-failed.",
	}, s.databaseSchema)
}

func (s *Server) databaseSchema(ctx context.Context, _ *sdk.CallToolRequest, in dbSchemaInput) (*sdk.CallToolResult, dbSchemaOutput, error) {
	conn, ok := db.Resolve(s.app)
	if !ok || conn == nil || conn.SQLDB() == nil {
		return failure("no SQL connection is registered"), dbSchemaOutput{}, nil
	}
	dialect := conn.Dialect()
	out := dbSchemaOutput{Dialect: string(dialect)}

	tableQuery, columnQuery := schemaQueries(dialect)
	if tableQuery == "" {
		return failure("listing tables is not implemented for dialect %q", dialect), out, nil
	}

	if table := strings.TrimSpace(in.Table); table != "" {
		rows, err := conn.SQLDB().QueryContext(ctx, columnQuery, table)
		if err != nil {
			return failure("describing %q: %v", table, err), out, nil
		}
		defer rows.Close()

		info := tableInfo{Name: table}
		for rows.Next() {
			var name, dataType, nullable string
			var deflt *string
			if err := rows.Scan(&name, &dataType, &nullable, &deflt); err != nil {
				return failure("reading columns of %q: %v", table, err), out, nil
			}
			column := columnInfo{
				Name:     name,
				Type:     dataType,
				Nullable: strings.EqualFold(nullable, "yes") || nullable == "1",
			}
			if deflt != nil {
				column.Default = *deflt
			}
			info.Columns = append(info.Columns, column)
		}
		if err := rows.Err(); err != nil {
			return failure("reading columns of %q: %v", table, err), out, nil
		}
		if len(info.Columns) == 0 {
			return failure("no table named %q", table), out, nil
		}
		out.Tables = []tableInfo{info}

		var b strings.Builder
		fmt.Fprintf(&b, "%s (%s)\n\n", table, dialect)
		for _, c := range info.Columns {
			null := "NOT NULL"
			if c.Nullable {
				null = "NULL"
			}
			fmt.Fprintf(&b, "  %-28s %-28s %s", c.Name, c.Type, null)
			if c.Default != "" {
				fmt.Fprintf(&b, " DEFAULT %s", c.Default)
			}
			b.WriteByte('\n')
		}
		return text(b.String()), out, nil
	}

	rows, err := conn.SQLDB().QueryContext(ctx, tableQuery)
	if err != nil {
		return failure("listing tables: %v", err), out, nil
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return failure("listing tables: %v", err), out, nil
		}
		out.Tables = append(out.Tables, tableInfo{Name: name})
	}
	if err := rows.Err(); err != nil {
		return failure("listing tables: %v", err), out, nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d tables (%s). Pass a table name for its columns.\n\n", len(out.Tables), dialect)
	for _, t := range out.Tables {
		fmt.Fprintf(&b, "  %s\n", t.Name)
	}
	return text(b.String()), out, nil
}

// schemaQueries returns the table-listing and column-describing SQL for a
// dialect, or empty strings when it is not supported.
//
// information_schema for Postgres and MySQL, because both have it and it is
// the same shape. SQLite does not, so it gets its own pair against
// sqlite_master and pragma_table_info — which is exactly why this is a
// per-dialect lookup rather than one "portable" query that silently returns
// nothing on one of them.
func schemaQueries(dialect db.Dialect) (tables, columns string) {
	switch dialect {
	case db.Postgres:
		return `SELECT table_name FROM information_schema.tables
                 WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
                 ORDER BY table_name`,
			`SELECT column_name, data_type, is_nullable, column_default
			   FROM information_schema.columns
			  WHERE table_schema = 'public' AND table_name = $1
			  ORDER BY ordinal_position`
	case db.MySQL:
		return `SELECT table_name FROM information_schema.tables
                 WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE'
                 ORDER BY table_name`,
			`SELECT column_name, column_type, is_nullable, column_default
			   FROM information_schema.columns
			  WHERE table_schema = DATABASE() AND table_name = ?
			  ORDER BY ordinal_position`
	case db.SQLite:
		return `SELECT name FROM sqlite_master
                 WHERE type = 'table' AND name NOT LIKE 'sqlite_%'
                 ORDER BY name`,
			`SELECT name, type, CASE WHEN "notnull" = 0 THEN 'YES' ELSE 'NO' END, dflt_value
			   FROM pragma_table_info(?)`
	}
	return "", ""
}

// --- database-query ---------------------------------------------------------

type dbQueryInput struct {
	SQL string `json:"sql" jsonschema:"one SQL statement"`
}

type dbQueryOutput struct {
	Columns   []string `json:"columns,omitempty"`
	Rows      [][]any  `json:"rows,omitempty"`
	RowCount  int      `json:"row_count"`
	Truncated bool     `json:"truncated"`
	Affected  *int64   `json:"rows_affected,omitempty"`
}

func (s *Server) addDatabaseQuery(server *sdk.Server) {
	description := "Run one SQL statement against this application's database and read the result. " +
		"Read-only: only SELECT, WITH, SHOW, EXPLAIN, TABLE and VALUES are accepted, and only " +
		"one statement per call."
	if s.allowWrites {
		description = "Run one SQL statement against this application's database. " +
			"This server was built with writes ENABLED, so statements that modify data " +
			"will be executed. Only one statement per call."
	}
	sdk.AddTool(server, &sdk.Tool{Name: "database-query", Description: description}, s.databaseQuery)
}

func (s *Server) databaseQuery(ctx context.Context, _ *sdk.CallToolRequest, in dbQueryInput) (*sdk.CallToolResult, dbQueryOutput, error) {
	conn, ok := db.Resolve(s.app)
	if !ok || conn == nil || conn.SQLDB() == nil {
		return failure("no SQL connection is registered"), dbQueryOutput{}, nil
	}
	sql := strings.TrimSpace(in.SQL)
	if sql == "" {
		return failure("no statement given"), dbQueryOutput{}, nil
	}
	if !s.allowWrites {
		if err := checkReadOnly(sql); err != nil {
			return failure("%v", err), dbQueryOutput{}, nil
		}
	}

	// A write still goes through Query rather than Exec where it returns rows
	// (RETURNING), so the two paths are decided by what came back, not by
	// guessing from the statement.
	rows, err := conn.SQLDB().QueryContext(ctx, sql)
	if err != nil {
		return failure("%v", err), dbQueryOutput{}, nil
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return failure("reading columns: %v", err), dbQueryOutput{}, nil
	}
	out := dbQueryOutput{Columns: columns}

	for rows.Next() {
		if len(out.Rows) >= s.maxRows {
			out.Truncated = true
			break
		}
		cells := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range cells {
			pointers[i] = &cells[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return failure("reading a row: %v", err), out, nil
		}
		for i, cell := range cells {
			// []byte is how a driver hands back text and it renders as a
			// number list in JSON, which is unreadable and looks like data
			// corruption to whoever is reading the transcript.
			if raw, ok := cell.([]byte); ok {
				cells[i] = string(raw)
			}
		}
		out.Rows = append(out.Rows, cells)
	}
	if err := rows.Err(); err != nil {
		return failure("reading rows: %v", err), out, nil
	}
	out.RowCount = len(out.Rows)

	var b strings.Builder
	if len(columns) == 0 {
		b.WriteString("the statement returned no columns\n")
		return text(b.String()), out, nil
	}
	b.WriteString(strings.Join(columns, " | "))
	b.WriteByte('\n')
	b.WriteString(strings.Repeat("-", len(strings.Join(columns, " | "))))
	b.WriteByte('\n')
	for _, row := range out.Rows {
		cells := make([]string, len(row))
		for i, cell := range row {
			cells[i] = fmt.Sprintf("%v", cell)
		}
		b.WriteString(strings.Join(cells, " | "))
		b.WriteByte('\n')
	}
	fmt.Fprintf(&b, "\n%d row(s)", out.RowCount)
	if out.Truncated {
		fmt.Fprintf(&b, ", truncated at %d — add LIMIT and OFFSET to page through", s.maxRows)
	}
	b.WriteByte('\n')
	return text(b.String()), out, nil
}

// --- migration-status -------------------------------------------------------

type migrationStatusOutput struct {
	Applied []string `json:"applied"`
	Count   int      `json:"count"`
}

func (s *Server) addMigrationStatus(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name: "migration-status",
		Description: "Which migrations this database has applied, newest first, read from " +
			"schema_migrations. Compare it against internal/migrations to see what is pending " +
			"before writing a new one — a version that collides with an applied one will not run.",
	}, s.migrationStatus)
}

func (s *Server) migrationStatus(ctx context.Context, _ *sdk.CallToolRequest, _ emptyInput) (*sdk.CallToolResult, migrationStatusOutput, error) {
	conn, ok := db.Resolve(s.app)
	if !ok || conn == nil || conn.SQLDB() == nil {
		return failure("no SQL connection is registered"), migrationStatusOutput{}, nil
	}

	rows, err := conn.SQLDB().QueryContext(ctx, `SELECT version FROM schema_migrations ORDER BY version DESC`)
	if err != nil {
		// The ordinary case for a database that has never been migrated, so
		// it is reported as that rather than as a failure.
		return text(fmt.Sprintf("could not read schema_migrations (%v); this database may never have been migrated", err)),
			migrationStatusOutput{}, nil
	}
	defer rows.Close()

	out := migrationStatusOutput{}
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return failure("reading schema_migrations: %v", err), out, nil
		}
		out.Applied = append(out.Applied, version)
	}
	if err := rows.Err(); err != nil {
		return failure("reading schema_migrations: %v", err), out, nil
	}
	out.Count = len(out.Applied)

	var b strings.Builder
	fmt.Fprintf(&b, "%d migrations applied, newest first:\n\n", out.Count)
	for _, version := range out.Applied {
		fmt.Fprintf(&b, "  %s\n", version)
	}
	return text(b.String()), out, nil
}
