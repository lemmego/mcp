## When to use this skill

When adding full-text search over tables an application already has.

This is not an indexing engine. The default engine searches PostgreSQL
`tsvector` columns, which is why `search` is offered on Postgres alone.

## Every table must declare `Visible`

A `Table` with no `Visible` predicate is refused with `ErrNoVisibility` rather
than quietly searching every row.

Search is where a forgotten visibility rule surfaces, because it is the one
read path that touches **every** row in a table rather than the ones somebody
navigated to. The safe default is unknowable from inside the module; the unsafe
one is "everything".

    Visible: "deleted_at IS NULL AND published_at IS NOT NULL",
    Visible: "(is_public OR owner_id = " + search.ViewerPlaceholder + ")",

`{{viewer}}` (`search.ViewerPlaceholder`) is replaced with the right
placeholder and bound to `Query.Viewer`. Use the marker rather than a literal
`$1`: Postgres **rejects a statement that binds a parameter it never
references**, so a fixed `$1` would force every viewer-independent predicate to
carry a no-op comparison.

It is a string rather than a structured filter because the rules that matter
are not column comparisons. "A post in a circle, unless the circle is private
and you are not in it" is an `EXISTS`, and reimplementing it as something
simpler would be a second copy of a rule that is hard to get right once.

## The query never throws, so do not pre-sanitise

`to_tsquery` raises a syntax error on input like `a & | b`, so a search box
wired to it answers 500 for a string somebody can type by accident. Everything
goes through `websearch_to_tsquery` — the only one of Postgres's four
conversion functions designed for untrusted input — which brings quoted
phrases, `or` and `-exclusion` with it.

Pass the user's string straight through. Stripping characters first removes
the phrase and exclusion syntax people expect.

`MaxQueryBytes` is 200; clamp rather than reject if you surface a limit.

## Only `Snippet` is HTML

`ts_headline` returns **the document** — somebody's bio, a comment — with
delimiters around the matches, so interpolating it raw publishes user content
as markup.

The module asks Postgres for `\x02` and `\x03`, two ASCII control characters
that are not HTML metacharacters, escapes the whole string, and introduces
`<mark>` afterwards. That ordering is the only one that works.

So: `Snippet` returns `template.HTML` and that promise is kept by that one
function. **`Title` and `Subtitle` stay plain strings** for the renderer to
escape like any other value — do not wrap them in `template.HTML` to "match".

**A headline with no match returns empty.** Postgres returns an opening
fragment when the term matched a column the snippet is not drawn from, usually
a title, and showing that as the matched text says "this is why it matched"
about text that is not. Render the subtitle when the snippet is empty.

## Results are grouped by source, never blended

`ts_rank` is a number about one document against one query, and it is **not
comparable across documents of different shapes**: a four-word product tagline
and a three-paragraph bio that match equally well get ranks an order of
magnitude apart. Merging them produces an ordering that looks authoritative
and is arbitrary.

`Results.Groups` is in the order the tables were passed, and a caller renders
them as sections. `Hit.Rank` is exposed because re-sorting **within** one
source is legitimate — do not sort across groups with it.

## Hydration

`ErrNoHydrator` means `Get` or `Each` was called before `Hydrate`. Search
returns identifiers and ranks; loading the rows is a separate step so that one
query loads them all rather than one per hit.

## Prefix matching and limits

Prefix terms are capped at `MaxPrefixTerms` (6), `DefaultPerSource` is 5 and
`MaxPerSource` is 100. `ClampLimit` applies the bounds — use it rather than
validating by hand.

## Engines

`EnginePostgres` is built in and registers itself. Meilisearch and the memory
engine live in `search/engines` and need a blank import. `ErrUnsupported` is
what an engine returns for an operation it does not have, rather than silently
doing something else.

## Check first

`search-docs` section `search`, `module-api` on `search`.
