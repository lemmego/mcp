package mcp

import (
	"strings"
	"testing"
)

// The guard is an allowlist, and these are the statements it allows.
func TestCheckReadOnlyAcceptsReads(t *testing.T) {
	accepted := []string{
		"SELECT 1",
		"select * from users",
		"  SELECT id FROM users WHERE id = 1  ",
		"WITH recent AS (SELECT 1) SELECT * FROM recent",
		"EXPLAIN SELECT 1",
		"EXPLAIN ANALYZE SELECT 1",
		"SHOW TABLES",
		"TABLE users",
		"VALUES (1), (2)",
		"SELECT 1;",
		"(SELECT 1)",
		"((SELECT 1))",
		"select\n  id\nfrom users",
	}
	for _, sql := range accepted {
		if err := checkReadOnly(sql); err != nil {
			t.Errorf("checkReadOnly(%q) = %v, want nil", sql, err)
		}
	}
}

// An allowlist rather than a denylist, because a denylist has to enumerate
// every way SQL can change something and will miss one. These are the ones
// that matter, including the obscure ones a denylist typically forgets.
func TestCheckReadOnlyRefusesEverythingElse(t *testing.T) {
	refused := []string{
		"UPDATE users SET name = 'x'",
		"DELETE FROM users",
		"INSERT INTO users (id) VALUES (1)",
		"DROP TABLE users",
		"TRUNCATE users",
		"ALTER TABLE users ADD COLUMN x int",
		"CREATE TABLE t (id int)",
		"GRANT ALL ON users TO someone",
		"COPY users FROM '/tmp/x'",
		"VACUUM FULL",
		"REINDEX TABLE users",
		"CLUSTER users",
		"COMMENT ON TABLE users IS 'x'",
		"SECURITY LABEL FOR x ON TABLE users IS 'y'",
		"DO $$ BEGIN PERFORM 1; END $$",
		"CALL some_procedure()",
		"SET search_path = public",
		"REFRESH MATERIALIZED VIEW v",
	}
	for _, sql := range refused {
		if err := checkReadOnly(sql); err == nil {
			t.Errorf("checkReadOnly(%q) = nil, want a refusal", sql)
		}
	}
}

// The bypass a head-only check has: the guard inspects the first statement, so
// without this `SELECT 1; DROP TABLE users` would pass it.
func TestCheckReadOnlyRefusesStackedStatements(t *testing.T) {
	stacked := []string{
		"SELECT 1; DROP TABLE users",
		"SELECT 1;DELETE FROM users",
		"select 1 ; update users set name='x'",
	}
	for _, sql := range stacked {
		err := checkReadOnly(sql)
		if err == nil {
			t.Fatalf("checkReadOnly(%q) = nil, want a refusal", sql)
		}
		if !strings.Contains(err.Error(), "one statement") {
			t.Errorf("checkReadOnly(%q) refused for the wrong reason: %v", sql, err)
		}
	}
}

// A trailing semicolon is ordinary, not an attempt to stack statements.
func TestCheckReadOnlyAllowsATrailingSemicolon(t *testing.T) {
	if err := checkReadOnly("SELECT 1;"); err != nil {
		t.Errorf("a trailing semicolon was refused: %v", err)
	}
	if err := checkReadOnly("SELECT 1;  "); err != nil {
		t.Errorf("a trailing semicolon with trailing space was refused: %v", err)
	}
}

// The refusal has to name the fix, or an agent will try to phrase its way
// around it instead of asking for the option to be enabled.
func TestReadOnlyRefusalNamesTheFix(t *testing.T) {
	err := checkReadOnly("DELETE FROM users")
	if err == nil {
		t.Fatal("expected a refusal")
	}
	if !strings.Contains(err.Error(), "WithWrites") {
		t.Errorf("the refusal does not name the option that lifts it: %v", err)
	}
}

// Writes being off by default is the whole safety posture, so it is asserted
// rather than assumed.
func TestWritesAreOffUnlessAskedFor(t *testing.T) {
	if (&Server{}).allowWrites {
		t.Error("a zero Server allows writes")
	}
	s := &Server{}
	WithWrites()(s)
	if !s.allowWrites {
		t.Error("WithWrites() did not enable writes")
	}
}

func TestWithMaxRowsIgnoresNonsense(t *testing.T) {
	s := &Server{maxRows: defaultMaxRows}
	WithMaxRows(0)(s)
	WithMaxRows(-5)(s)
	if s.maxRows != defaultMaxRows {
		t.Errorf("maxRows = %d after nonsense input, want the default %d", s.maxRows, defaultMaxRows)
	}
	WithMaxRows(10)(s)
	if s.maxRows != 10 {
		t.Errorf("maxRows = %d, want 10", s.maxRows)
	}
}
