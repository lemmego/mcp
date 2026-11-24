## When to use this skill

Anything multi-tenant: resolving which tenant a request is about, scoping a
query, or setting up row-level security. Read it before writing tenant scoping
by hand — this module exists and does it properly.

## A header selects. It does not authorise.

This is the part worth reading twice. A header is client-supplied, so on its
own it says nothing about whether the caller may see that tenant. Taking the
tenant from a header and scoping the query to it — with nothing else — means
**any authenticated member of one tenant can read another by typing its id**,
and both the ORM predicate and the row-level security policy will enforce that
choice perfectly. The isolation machinery is not what protects you here.

So `Identify` answers only "which tenant is this request *about*". The binding
comes from the caller's **membership**:

    binding, err := tenancy.Authorize(ctx, registry, identification, principal)

A caller who names a tenant they do not belong to gets `ErrNoTenant` — the same
error as a tenant that does not exist. That is deliberate: a distinct
"forbidden" would confirm which companies are customers and let anybody
enumerate the tenant list from a sign-in page. **Do not "improve" that error.**

`Select` and `Confirm` are the two halves, for the case that forces them apart:
a user lookup needs a tenant (the users table is scoped), and authorising needs
the user that lookup produces. Resolve the host first, authorise after, with
nothing bound in between.

## Two checks to run at boot, in every environment

    rls.AssertNotPrivileged(ctx, db)   // package function
    cfg.SelfTest(ctx, db)              // METHOD on rls.Config, not a package function

The first matters because **a superuser bypasses row-level security entirely,
and `FORCE` does not change that** — `FORCE` only subjects the table's owner. A
developer's PostgreSQL role is usually a superuser, so without this check the
policies are inert locally and every isolation test passes while proving
nothing.

Note the shapes, because they differ: `AssertNotPrivileged` is a **package
function** taking a `*sql.DB`, while `SelfTest` is a **method on
`rls.Config`**. Ask `module-api` about `tenancy/rls` rather than guessing —
there is no `tenancy/security` package.

The second turns the one infrastructure choice that would silently destroy the
design — a connection pooler in *statement* mode — into a startup error rather
than a cross-tenant leak under load.

Both go in every environment, not just production. Local is where they catch
things.

## What this package deliberately does not do

- **It does not know what a user is.** `Principal` is two strings.
- **It does not cache.** `Lookup` runs per request. A cache belongs in front of
  the `Registry`, **keyed per source** — conflating a host-resolved and a
  header-resolved entry means one can poison the other, which is a tenant leak
  rather than a cache bug. `Binding.Source` exists so you can key them apart.
- **It does not filter `Raw` SQL or many-to-many pivot reads.** Neither does
  the ORM: there is nowhere to inject a predicate into hand-written SQL, and a
  pivot is a bare table with no model and so no declared tenant column. Both
  are row-level security's job, which is why RLS is not optional.

## The realistic failure mode

It is not a broken policy. It is **a table that gains a tenant column, no
policy, and nobody notices** — tests pass, the page works, and a customer finds
it. Run the conformance check the module ships rather than trusting review.

## One operational limit worth knowing

A tenant binding is attached by the transaction, and this module holds a
transaction open for the request. On libSQL/sqld a transaction cannot sit
**idle** for more than about ten seconds before the stream expires, and the
client reports it as an empty error message. So: **no outbound HTTP inside the
tenant transaction.** On PostgreSQL this is a held connection rather than a
hard limit, but the same discipline applies.

## Check first

There is **no documentation page for `tenancy` on the site yet** — the module's
README is the authoritative spec, and it is detailed. Use `module-api` for
signatures.
