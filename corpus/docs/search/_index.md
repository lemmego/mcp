---
title: Search
type: docs
prev: docs/media/
next: docs/seo/
weight: 132
sidebar:
  open: true
---

`lemmego/search` adds full-text search over Postgres `tsvector` columns your
tables already have. It turns what somebody typed into a safe query, and
returns results grouped by source with highlighted snippets.

**The default engine holds no index of its own**: it queries generated
`tsvector` columns and GIN indexes that your migrations created. If your tables
do not have them yet, add them first — it is cheap in a new migration and an
expensive retrofit on a large live table.

That is the default and not the only option. The engine is pluggable, and
Meilisearch ships alongside Postgres; see [Choosing an
engine](#choosing-an-engine) for what changes when you switch.

```go
import "github.com/lemmego/search"

profiles := search.Table{
    Name: "profiles", Label: "People",
    From: "users", Vector: "search_vector",
    ID: "id", Title: "name", Subtitle: "headline",
    URL: "'/@' || handle",
    Snippet: []string{"headline", "bio"},
    Visible: "deleted_at IS NULL AND profile_published",
}

results, err := search.New(db).Search(ctx,
    search.Query{Text: "postgres", Viewer: userID}, profiles, launches)
```

## Install

```shell
go get github.com/lemmego/search
```

There is no provider to register. `New` takes anything with `QueryContext` and
`QueryRowContext`, so `*sql.DB`, `*sql.Tx` and `*sql.Conn` all work, and the
module depends on no ORM:

```go
search.New(app.Get[*orm.DB](a).SQLDB())   // lemmego/orm
search.New(gormDB.DB())                    // GORM
```

## Behaviour you can rely on

Four properties are worth knowing before you build a page on this, because
each one rules out a common way search goes wrong.

### Bad input never errors

`to_tsquery` raises a **syntax error** on input like `a & | b`, so a search box
wired to it answers 500 for a string somebody can type by accident. Everything
goes through `websearch_to_tsquery`, the only one of Postgres's four
conversion functions designed for untrusted input — and it brings quoted
phrases, `or`, and `-exclusion` with it.

Hostile and malformed input — including `'; DROP TABLE x; --` — returns no
results rather than an error.

### Snippets are escaped before they are highlighted

{{< callout type="warning" >}}
`ts_headline` returns **the document** — somebody's bio, a comment, a job
description — with delimiters around the matches. Interpolating that into a
page publishes user content as markup by construction.
{{< /callout >}}

So the delimiters requested from Postgres are `\x02` and `\x03`, two ASCII
control characters. Neither is an HTML metacharacter, so the whole string is
escaped first and the `<mark>` elements are substituted in afterwards. Asking
Postgres for `<mark>` directly would mix the markup and the document, leaving
no way to escape one without the other.

`Snippet` returns `template.HTML`, and it is the only field that does.
`Title` and `Subtitle` are plain strings for your renderer to escape like any
other value.

**A headline with no match in it returns empty.** Postgres returns an opening
fragment when the term matched a column the snippet is not drawn from — a
title, usually — and presenting that as the matched text says "this is why it
matched" about text that is not why it matched.

### Results are grouped, never blended

`ts_rank` is a number about one document against one query. It is **not
comparable across documents of different shapes**: a four-word product tagline
and a three-paragraph bio that match equally well get ranks an order of
magnitude apart. Merging them produces an ordering that looks authoritative
and is arbitrary.

`Results.Groups` comes back in the order you passed the tables, so render them
as sections. `Hit.Rank` is exposed for re-sorting *within* one source, which is
valid.

### Visibility is required

A `Table` with no `Visible` predicate is refused with `ErrNoVisibility` rather
than quietly searching every row.

Search is where a forgotten visibility rule shows up, because it is the one
read path that touches every row in a table rather than the rows somebody
navigated to. This package cannot know your safe default, and the unsafe one
is "everything".

```go
Visible: "deleted_at IS NULL AND published_at IS NOT NULL",
Visible: "(is_public OR owner_id = " + search.ViewerPlaceholder + ")",
```

`{{viewer}}` (`ViewerPlaceholder`) is substituted with the correct placeholder
and bound to `Query.Viewer`. It is a named marker rather than a fixed `$1`
because Postgres rejects a statement that binds a parameter it never
references, so a predicate that does not mention the viewer would otherwise
need a no-op comparison to be bindable.

## `Table`

| Field | |
|---|---|
| `Name`, `Label` | identify and caption the group |
| `From` | the relation, optionally with a join: `"job_posts j JOIN companies c ON …"` |
| `Vector` | the tsvector column — see below |
| `Config` | the text search configuration, default `"english"` |
| `ID`, `Title`, `Subtitle` | column expressions selected into the `Hit` |
| `URL` | an expression for the address: `"'/launch/' \|\| slug"` |
| `Snippet` | columns `ts_headline` draws from, concatenated |
| `Visible` | **required** |
| `Tiebreak` | orders equal ranks, default `ID DESC` |

Every field here is written by you and interpolated into the statement,
because an identifier cannot be a bind parameter. **What somebody typed is
always a parameter.** Keeping those two apart is why a `Table` is a
declarative descriptor you define at startup rather than a builder that takes
strings per request.

{{< callout type="warning" >}}
**The predicate is `<Vector> @@ <query>` against the bare column**, because
that is the form the GIN index answers. Wrapping it in anything — a
`coalesce`, a cast, a `to_tsvector` of it — silently turns every search into a
sequential scan.
{{< /callout >}}

**A column built from more than one configuration** — `simple` for a handle
and `english` for a bio, which is reasonable — can only be queried through one
of them, so a term that stems differently in the two will not match the half it
was not parsed for.

## Prefix matching

```go
search.Query{Text: "postg", Prefix: true}   // matches "postgres"
```

Off by default, because it changes what a query means: with it, `go` matches
`golang` and `gorilla` — right for type-ahead, wrong once somebody has pressed
enter.

Prefix matching needs the `:*` operator, which only `to_tsquery` understands.
Nothing a person typed reaches that function directly: every rune that is not
a letter or digit becomes a separator, the expression is assembled from terms
this package produced, and each term is still bound as a parameter.

## Performance

**One statement per table, plus a count.** Not a `UNION ALL`, because the
ranks are not comparable so merging in SQL gains nothing — and not concurrent,
because each query is an index scan answering in single-digit milliseconds,
and fanning eight out over the pool would cost a connection per source under
load.

**`ts_headline` is the expensive part** — it re-parses the document. It sits in
the select list of a query that `LIMIT`s, so it runs once per returned row and
never over the matched set.

**The count is a separate query**, not a window function. `COUNT(*)` on the
same predicate is answered from the index; a window function makes every page
of a paginated search recompute the total.

**Input is bounded**: 200 bytes of query, 6 prefix terms, `MaxPerSource` of
100.

## Empty and unsearchable input

```go
if !search.Searchable(text) { /* "type something" */ }
```

Use it to tell "nothing was typed" apart from "nothing matched". Punctuation
alone parses to an empty `tsquery`, which matches nothing, so a page would
otherwise report "no results" for a question it never asked. `Search` sets
`Results.Empty` and runs no queries at all for such input.

## Testing

```bash
go test ./...                                      # query building and escaping
SEARCH_POSTGRES_DSN="host=localhost dbname=postgres sslmode=disable" \
  go test ./...                                    # and the SQL
```

{{< callout type="warning" >}}
The Postgres tests **skip** without a DSN rather than fail, so a green run
proves less than it looks. Set `SEARCH_POSTGRES_DSN` when changing anything
here: the unit tests never execute the generated SQL, so a syntax error passes
all of them.
{{< /callout >}}

## Choosing an engine

An engine is selected from configuration and built once:

```go
engine, err := search.Build(ctx, &search.Config{
    Engine:   search.EngineMeili,          // or EnginePostgres, the default
    Prefix:   "myapp_prod",
    Meili:    search.MeiliConfig{Host: "http://127.0.0.1:7700", APIKey: key},
})
```

An empty `Engine` means `postgres`, because that is the one engine needing no
infrastructure your application does not already have. `Postgres.DB` takes the
application's own handle rather than a DSN, so there is one pool with one set
of limits.

`Prefix` namespaces every index an engine writes, and it is what bounds
`Flush`. An instance shared between environments wants one: without it, a
`search:flush posts` run from a laptop empties production's `posts` index.

Engines that are not built in register themselves when their package is
imported, the same arrangement as [a filesystem
driver](/docs/filesystem/#a-cloud-disk-needs-one-blank-import):

```go
import _ "github.com/lemmego/search/engines"
```

Postgres is deliberately not in there — it needs nothing but `database/sql`, so
it is in the root and registers itself. Without the import, `Build` says so:

```text
search: no engine registered for "meilisearch" (registered: [postgres]);
import github.com/lemmego/search/engines to register the ones that are not
built in
```

### What changes when you switch

Engines are not interchangeable, and `Capabilities()` is how one says so
rather than pretending:

| | Postgres | an external engine |
|---|---|---|
| `RelationalVisibility` | yes — `Visible` is SQL over your own rows | no |
| `FilterVisibility` | no | yes — an expression over indexed fields |
| `ExactTotals` | yes, `Group.Total` is a count | no, it is an estimate |

The first row is the one with teeth. A `Visible` rule like three nested
`EXISTS` over `follows` and `circle_members` is a live join against mutable
tables, and no external engine's filter language can evaluate it. So an engine
that cannot enforce a source's visibility **refuses that source** — before any
query runs, rather than searching rows it has no way to filter and handing
somebody's private post to a stranger.

`ExactTotals` matters at the other end: a page rendering "23 results" from an
estimate is asserting something it was not told.

### Keeping an index fed

An engine that holds its own index needs documents put into it. A model
implements `Indexable` — `SearchIndex()`, `SearchKey()` and
`SearchDocument()` — and the engine takes them through `Index`, `Unindex` and
`Flush`.

`SearchDocument` returns only what search needs. Every field in it leaves your
database, lands in another system's storage, and comes back to anyone who can
reach that system, so a document carrying a password hash because the struct
had one is a disclosure with no request behind it.

## Limitations

No synonyms, no spelling correction, no faceting and no aggregations.

The `Table` descriptors on this page are Postgres-shaped and do not carry over
to another engine, which takes its own descriptor type. `Query`, `Hit`, `Group`
and `Results` are shared and do carry over.

## Next steps

- [Raw SQL and DDL](/docs/database/raw-sql-and-ddl) — adding the generated
  column and GIN index a `Table` queries.
- [Migrations](/docs/migrations/) — where that DDL belongs.
