## When to use this skill

When deciding whether *this* user may do *this* to *this record*. Roles and
permissions that are not about a specific record are `rbac`; who the user is at
all is `auth`.

## Two calls, and they are not interchangeable

    // In a handler: Authorize, because the reason matters.
    if err := authz.Authorize(c, policies.ProfileUpdate, target); err != nil {
        return err
    }

    // In a view: Can, because you are deciding whether to render a button.
    canEdit := authz.Can(c, policies.ProfileUpdate, target)

**Four functions, two shapes.** `Authorize` and `Can` are generic over the
record and are for a policy about a particular row. `AuthorizeGate(c,
ability)` and `Allowed(c, ability)` are the resourceless pair, for an ability
with no record attached — which is what a permission code from `rbac` is.
Passing `nil` to `Can` does not compile: it cannot infer the type parameter.

`Authorize` returns an error carrying the policy's own reason, and returning it
gives the reader the sentence the policy wrote rather than a bare 403 they
cannot act on. `Can` throws that away for a boolean.

Load the record first and ask once. A handler that asks about two records to
decide one thing is usually missing a policy.

## An ability nobody registered is denied

`authz` refuses an unregistered ability before any `Before` hook runs. That is
deliberate and has a consequence worth knowing: **a typo in an ability name is
a denial, not a silent pass** — including for somebody who is supposed to be
allowed everything. If a permission check is failing for an administrator, look
for the missing registration before looking at the role.

## Policies are registered once, at boot

    &authz.Provider{Policies: func(r *authz.Registry) {
        r.Register(policies.ProfileUpdate, policies.CanUpdateProfile)
    }}

The provider goes in `LoadProviders`, after authentication — a policy compares
a record's owner against the signed-in user, and without one every policy would
be asked about nobody and would have to answer no, which is a working module
that refuses everything.

## The two hook shapes

- A **`Gate`** answers one named ability. One gate per ability code.
- **`Before`** is handed the subject and **not** the ability, so it cannot
  implement per-ability checks — but it is the only hook that runs for a
  resource policy registered with `Register`.

That asymmetry is why `rbac` installs both; see the `lemmego-rbac` skill before
wiring roles into this.

## Check first

`module-api` on `authz` for signatures, `search-docs` section `authz` for the
walkthrough. In a generated project, `internal/routes/authz.go` and
`internal/policies/` hold a worked example — read those before inventing a
second shape beside them.
