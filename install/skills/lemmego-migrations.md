## When to use this skill

When adding, editing or rolling back a migration in a Lemmego project.

## Check first

Run `migration-status` before writing one. A version that collides with an
applied migration will not run, and the collision is silent.

## The schema builder's sharp edges on Postgres

Each of these was found by shipping it, so prefer raw `tx.Exec` whenever the
exact DDL matters:

- **Multi-column index names carry no table prefix**, and Postgres index names
  are schema-global — so two tables both wanting `(user_id, position)` collide
  and the second fails the batch. Write multi-column indexes as raw SQL.
- **`Enum` emits no CHECK constraint at all** on Postgres, silently. Use
  `String` plus a raw CHECK.
- **`ForeignID(col)` adds a BIGSERIAL**, giving the column its own sequence and
  an implicit default. Use `UnsignedBigInt` plus `Foreign(col).References(...)`.
- **`TimestampTz(name, p)` with p > 0 emits invalid SQL** — always pass 0.
- **A database DEFAULT never applies through the ORM**, which names every mapped
  column in its INSERT, so a `DEFAULT true` bool is written as Go's zero value.
  Set it in `BeforeCreate` as well.

## The whole batch is one transaction

With a hardcoded 30s timeout, so `CREATE INDEX CONCURRENTLY` is impossible and
the batch is all-or-nothing. Postgres DDL is transactional, so functions and
triggers are fine.

## Never migrate through a connection pooler

`migrate` resolves `--dsn`, then `DATABASE_URL_UNPOOLED`, then `DATABASE_URL`, then the
`DB_*` parts. Managed providers publish a pooled URL and point you at it; under
transaction pooling lib/pq's unnamed prepared statements fail intermittently as
`bind message supplies N parameters, but prepared statement "" requires M`.

## Verify reversibility

Up, down, and up again on a throwaway database. That catches a down migration
that leaves a table behind and an up that only works on an empty schema.
