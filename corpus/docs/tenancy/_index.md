---
title: Multi-tenancy
type: docs
prev: docs/rbac/
next: docs/ratelimit/
weight: 108
sidebar:
  open: true
---

`lemmego/tenancy` answers two questions for every request: which tenant is this
about, and may this caller see it. Reach for it when one deployment serves
several customers whose rows must never meet — a workspace product, an agency
tool, anything sold per organisation.

A tenant here is a row in your own table. The module resolves which one a
request names, checks the caller belongs to it, and opens a database
transaction bound to it so your queries are scoped without every call site
remembering to say so.

## Install

```shell
go get github.com/lemmego/tenancy
```

| Package | Holds |
| --- | --- |
| `tenancy` | resolving a tenant and authorising a caller |
| `tenancy/provider` | the request-path wiring |
| `tenancy/hostname` | the domain layout: which host is which surface |
| `tenancy/sqlstore` | a `Registry` backed by a table |
| `tenancy/rls` | PostgreSQL row-level security |

It needs the [ORM](/docs/orm/) and, for row-level security, PostgreSQL.

## A header selects. It does not authorise.

This is the part to read twice. A header is client-supplied, so on its own it
says nothing about whether the caller may see that tenant. Taking the tenant
from a header and scoping the query to it — with nothing else — lets any
authenticated member of one tenant read another by typing its id, and both the
ORM predicate and the row-level security policy will enforce that choice
perfectly. The isolation machinery does not protect you here.

So the two steps are separate:

```go
ident, err := tenancy.Identify(r, cfg)                 // which tenant is this about
binding, err := tenancy.Authorize(ctx, reg, ident, principal)  // may they see it
```

`Identify` reads the host, the header or the path. `Authorize` asks your
`Registry` whether the principal is a member, and a caller who names a tenant
they do not belong to gets `tenancy.ErrNoTenant` — the same error as a tenant
that does not exist. One error for both, so a sign-in page cannot be used to
enumerate which companies are customers.

Where a user lookup itself needs a tenant — the users table is scoped, so
finding the user requires knowing the tenant — use the two halves. `Select`
resolves the tenant from the request, and `Confirm` authorises once you have
the principal that lookup produced. Nothing is bound in between.

## Register it

```go
&provider.Provider{
    Config:    cfg,                 // the domain layout and header names
    Registry:  sqlstore.New(db, sqlstore.Config{}),
    DB:        ormDB,
    Principal: func(c app.Context) (tenancy.Principal, bool) {
        user, ok := auth.UserAs[*models.User](c)
        return user, ok
    },
    SkipPrefixes: []string{"/assets", "/health"},
}
```

`Principal` is supplied by the application, because this module must not learn
what a session is. `SkipPrefixes` are paths that never need a tenant — built
assets, health checks — and skipping them avoids a transaction per request for
something that reads no rows.

A zero `Config.Hostnames.AppDomain` is refused rather than defaulted: resolving
every host to one surface because nobody configured a domain is not a sensible
default.

## Read the current tenant

```go
tenant, ok := provider.Current(ctx)
tx, ok := provider.Tx(ctx)
```

`Tx` is the transaction the tenant is bound to. Queries run on it are scoped;
queries run on the bare handle are not.

## Supply the registry

`Registry` is two methods, so you can back it with anything:

```go
type Registry interface {
    Lookup(ctx context.Context, id Identifier) (Tenant, error)
    IsMember(ctx context.Context, tenantID string, principal Principal) (bool, error)
}
```

`sqlstore.New` implements it over a table — `tenants` by default — and takes a
`MemberFunc` for the membership query, since only the application knows where
memberships live.

## Row-level security

On PostgreSQL, the ORM's predicate is not the only line of defence. `rls`
generates the policies and sets a transaction-local setting the policies read:

```go
cfg := rls.Config{Setting: "app.tenant_id", Column: "tenant_id"}
```

Run two checks at boot, in every environment:

```go
if err := rls.AssertNotPrivileged(ctx, db); err != nil {  // package function
    return err
}
if err := cfg.SelfTest(ctx, db); err != nil {              // method on Config
    return err
}
```

`AssertNotPrivileged` matters because **a superuser bypasses row-level security
entirely, and `FORCE` does not change that** — `FORCE` only subjects the
table's owner. A development PostgreSQL role is usually a superuser, so without
this check the policies are inert locally and an isolation test passes while
proving nothing.

`SelfTest` confirms the setting really is transaction-local. A connection
pooler in *statement* mode makes it session-level, which leaks the tenant
across requests; the check turns that into a startup error and returns
`rls.ErrNotIsolated`.

## Behaviour and limits

- **`Lookup` runs per request.** There is no cache. One belongs in front of
  your `Registry`, **keyed per source** — conflating a host-resolved and a
  header-resolved entry means one can poison the other, which is a tenant leak
  rather than a cache miss. `Binding.Source` is there to key them apart.
- **Raw SQL and many-to-many pivot reads are not scoped.** There is nowhere to
  inject a predicate into hand-written SQL, and a pivot is a bare table with no
  model and so no declared tenant column. Both are row-level security's job,
  which is why it is not optional on PostgreSQL.
- **A tenant is bound by the transaction**, which the provider holds for the
  request. Avoid outbound HTTP inside it: on a server that expires idle
  transactions the stream closes and the next query fails with no useful
  message.
- **A request naming two different tenants** — a host and a header that
  disagree — is `tenancy.ErrAmbiguous`, not a silent preference for one.
- **The realistic failure is a table that gains a tenant column and no
  policy.** Tests pass, the page works, and a customer finds it. Run the
  conformance check rather than relying on review.

## Next steps

- [Roles and permissions](/docs/rbac/) — scoped per tenant by default, with the
  tenant read off the same transaction.
- [The ORM](/docs/orm/) — the predicate that scopes a query, and the
  transaction the tenant binds to.
- [Authentication](/docs/security/authentication/) — the principal whose
  membership authorises the binding.
