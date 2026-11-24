## When to use this skill

When adding roles or permissions, or wiring `rbac` into `authz`. `rbac` answers
"may this subject do this named thing at all"; `authz` answers "to this
record". Most applications need both.

## A scope is an opaque string, and the default is deliberate

Every role, grant and assignment belongs to a scope, and this module never asks
what one is:

    // Multi-tenant: the scope is the tenant the ORM transaction is bound to.
    store := rbacstore.New(rbacstore.Config{})

    // Single-tenant: one scope, always.
    store := rbacstore.New(rbacstore.Config{ScopeOf: rbacstore.Unscoped("")})

**Do not "simplify" that default.** The other default — one scope for everybody
— means a multi-tenant application that forgot to configure it shares one set
of roles across every tenant, silently, with every test passing. A
single-tenant application that has not configured it instead gets `ErrNoScope`
on its first call, which is a loud failure on day one.

A subject is opaque too: a user id, a service account, an API key, or a
qualified pair like `user:42` when administrators live in a different table
from customers.

## The scope comes off the handle, never from an argument

    func (h *Handler) Grant(c app.Context) error {
        return h.db.TenantTx(c.RequestContext(), workspaceID, func(tx *orm.Tx) error {
            return h.roles.Grant(c.RequestContext(), tx, roleID, "post.publish")
        })
    }

There is no spelling of these calls that could name a scope other than the one
the transaction is in. If you find yourself wanting to pass a scope, you want a
different transaction — a scope parameter is one more thing a caller can get
wrong in the one place where getting it wrong is a cross-tenant write.

## Both authz hooks are needed

    &authz.Provider{Policies: func(r *authz.Registry) {
        rbacauthz.Install[*models.User](r, catalog, rbacauthz.Options{})
    }}

`Install` registers **one `authz.Gate` per declared permission code** plus
**one `authz.Before`** for the system role. Installing only the gates is the
mistake to avoid: a member of the system role would pass every permission gate
and still be refused by an ownership policy on somebody else's row, which is
the opposite of what "every permission" means.

## Two more things that surprise people

- **The system role holds no grants at all.** It is a `Before` hook, not a role
  with every permission attached — so a permission added later is covered
  without a backfill.
- **Syncing the catalogue never deletes.** A code removed from the declared
  catalogue stays in the database. Removing a grant is a migration you write
  deliberately, because a sync that deleted would silently revoke access on
  the next deploy that renamed a code.

## Check first

There is **no documentation page for `rbac` on the site yet** — read the
module's README, which is the authoritative spec, and `module-api` for
signatures. The schema is yours: `rbac` expects certain columns to exist with
compatible types but does not own the migration.
