---
title: Roles and Permissions
type: docs
prev: docs/authz/
next: docs/tenancy/
weight: 107
sidebar:
  open: true
---

`lemmego/rbac` stores roles and permissions in your database and checks them
through [`authz`](/docs/authz/). Reach for it when an administrator needs to
create a role at runtime, tick a permission in a form, or change who can do
what without a deploy.

`authz` on its own gives you typed policies: Go code, reviewable in one place,
changed by editing and shipping. This gives you the part a policy cannot be — a
set of permissions that lives in rows. Most applications want both, and they
compose rather than compete.

## Install

```shell
go get github.com/lemmego/rbac
```

Three packages, and you will use all three:

| Package | Holds |
| --- | --- |
| `rbac` | the catalogue of permissions this build declares |
| `rbac/rbacstore` | the tables, and reads and writes against them |
| `rbac/rbacauthz` | the two hooks that make `authz` ask about them |

It needs [`authz`](/docs/authz/) and the [ORM](/docs/orm/).

## Declare the permissions this build has

A catalogue is a value your application builds, not a registry filled by
`init`:

```go
catalog := rbac.NewCatalog()
err := catalog.Add(
    rbac.Permission{Code: "post.create", Module: "post", Name: "Create posts"},
    rbac.Permission{Code: "post.publish", Module: "post", Name: "Publish posts"},
)
```

A code is `module.action`, lower case, and `Module` must be the first segment —
both are checked when you add it, so a typo is an error at boot rather than a
permission nobody can be granted.

## Register the store and the hooks

```go
func LoadProviders() []app.Provider {
    store := rbacstore.New(rbacstore.Config{})

    return []app.Provider{
        &ormconnector.Provider{},
        &auth.Provider{ /* … */ },
        &authz.Provider{Policies: func(r *authz.Registry) {
            rbacauthz.Install[*models.User](r, catalog, rbacauthz.Options{})
        }},
    }
}
```

`Install` registers one `authz` gate per declared permission, plus one
`Before` hook for the system role. Both are needed: the gates make each code
checkable, and the `Before` hook is the only one that runs for a record policy
registered with `authz.Register`. With the gates alone, a member of the system
role passes every permission check and is still refused by an ownership policy
on somebody else's row.

The provider goes after authentication, because a permission set is loaded for
whoever is signed in.

## Check a permission

```go
if err := authz.AuthorizeGate(c, "post.publish"); err != nil {
    return err
}
```

Or `authz.Allowed(c, "post.publish")` for the boolean, when you are deciding
whether to render a button.

These are the resourceless forms, and they are the ones a permission code
wants. `Authorize` and `Can` are generic over the record being acted on, so
they are for a policy about a particular row — see
[Authorization](/docs/authz/).

**A code nobody declared is denied.** `authz` refuses an unregistered ability
before any hook runs, so a misspelled check fails closed — including for
somebody who is supposed to be allowed everything.

## Scopes

Every role, grant and assignment belongs to a scope, and `rbac` never asks what
one is. It is an opaque string, so it can be a tenant, a workspace, an
organisation, or nothing at all.

```go
// The scope is the tenant the ORM transaction is bound to.
store := rbacstore.New(rbacstore.Config{})

// One scope for the whole application.
store := rbacstore.New(rbacstore.Config{ScopeOf: rbacstore.Unscoped("")})
```

The default reads the tenant from the transaction. A single-tenant application
that has not configured `ScopeOf` gets `rbacstore.ErrNoScope` on its first
call, which is the failure you want on the first run rather than one set of
roles quietly shared by every tenant.

A subject is opaque too — an integer id, a UUID, or a qualified pair like
`user:42` when administrators live in a different table from customers.

## Manage roles

Every write takes the transaction, and the scope comes off it:

```go
return h.db.TenantTx(c.Ctx(), workspaceID, func(tx *orm.Tx) error {
    role, err := store.CreateRole(c.Ctx(), tx, "editor", "Can publish posts")
    if err != nil {
        return err
    }
    if err := store.Grant(c.Ctx(), tx, role.ID, "post.publish"); err != nil {
        return err
    }
    return store.Assign(c.Ctx(), tx, subject, role.ID)
})
```

There is no argument that could name a scope other than the transaction's. The
rest of the surface is `Revoke`, `Unassign`, `DeleteRole`, `Roles`,
`RoleNamed`, `Grants` and `PermissionsOf`.

## The schema is yours

`rbacstore` creates no tables. It hands you the statements, and you put them in
a migration:

```go
for _, statement := range rbacstore.SchemaStatements(rbacstore.SchemaConfig{
    Config:            cfg,
    ScopeType:         "uuid",
    ScopeReferences:   "workspaces (id)",
    SubjectReferences: "users (workspace_id, id)",
}) {
    if _, err := tx.Exec(statement); err != nil {
        return err
    }
}
```

`DropStatements` is the reverse, for the migration's `Down`. Table and column
names come from `Config`.

## Behaviour and limits

- **The system role holds no grants.** It is the `Before` hook, so a
  permission added later is covered without backfilling any rows. Its name
  defaults to `superuser`.
- **Syncing the catalogue never deletes.** `SyncPermissions` adds codes the
  build declares and leaves the rest, so a renamed code does not revoke access
  on deploy. Removing one is a migration you write.
- **The permission set is loaded once per request** and reaches the gates
  through the context. `rbacauthz.SetFrom` reads it back; `WithSet` is how a
  job or a test puts one there.
- **Nothing from a request reaches an identifier.** Table and column names are
  interpolated once at boot, from your configuration.

## Next steps

- [Authorization](/docs/authz/) — the policies these permissions sit beside,
  and the difference between `Can` and `Authorize`.
- [Authentication](/docs/security/authentication/) — the signed-in user a
  permission set is loaded for.
- [Caching](/docs/cache/) — where a permission set is held between requests.
