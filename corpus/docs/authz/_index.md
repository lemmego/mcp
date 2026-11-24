---
title: Authorization
type: docs
prev: docs/session
next: docs/rbac/
weight: 106
sidebar:
  open: true
---

`lemmego/authz` decides what a request is allowed to do. You register typed
policies once at boot and ask them per request.

It is separate from [`lemmego/auth`](/docs/security/authentication), which
establishes *who* is asking. Authorization answers *what they may do*, so an
application can authenticate without authorizing anything, and a policy can be
written against a request that something other than `lemmego/auth`
authenticated.

## Install

```shell
go get github.com/lemmego/authz
```

```go
func LoadProviders() []app.Provider {
    return []app.Provider{
        &ormconnector.Provider{},
        &authz.Provider{Policies: policies.Register},
    }
}
```

`Policies` is called once at boot with the registry, so every policy in the
application is declared in one reviewable place:

```go
// internal/policies/policies.go
func Register(r *authz.Registry) {
    authz.Register(r, "launch.update", launchOwner)
    authz.Register(r, "comment.delete", commentOwnerOrModerator)
    authz.Gate(r, "launch.create", verifiedEmail)
}
```

## Write a policy

A policy is a function taking the context, the **typed** user and the
**typed** resource:

```go
func launchOwner(c app.Context, u *models.User, l *models.Launch) authz.Decision {
    if l.UserID == u.ID {
        return authz.Allow()
    }
    return authz.Deny("only the author can edit a launch")
}

authz.Register(r, "launch.update", launchOwner)
```

Type parameters are inferred from the function signature, and the registry
stores the types alongside the policy.

The reason on a `Deny` travels with the decision, so a 403 can say something
specific instead of just "Forbidden".

## Check it in a handler

```go
if err := authz.Authorize(c, "launch.update", launch); err != nil {
    return err
}
```

The returned error carries its own status, which the framework's error mapping
turns into **403** — or **401** when nobody is signed in, so an anonymous
visitor is prompted to sign in rather than told they are forbidden.

## Resource abilities and gates

|  | Register with | Ask with | Answers |
|---|---|---|---|
| **Resource** | `Register[U, R]` | `Can`, `Explain`, `Authorize` | "may you edit *this* launch" |
| **Gate** | `Gate[U]` | `Allowed`, `ExplainGate`, `AuthorizeGate` | "may you create launches at all" |

The two are not interchangeable. Asking a resource ability through
`AuthorizeGate` is refused rather than passed a nil resource, because a policy
written to inspect a resource and handed none would answer for the wrong
reason.

### Ways to ask

| Call | Returns | Use when |
|---|---|---|
| `Can(c, ability, res)` | `bool` | Deciding whether to render a button |
| `Explain(c, ability, res)` | `Decision` | You want the reason |
| `Authorize(c, ability, res)` | `error` | Guarding a write; return it |
| `Allowed(c, ability)` | `bool` | Gate, in a template |
| `AuthorizeGate(c, ability)` | `error` | Gate, guarding a write |

`Can` never fails the request, which makes it safe in a props builder:

```go
props["can_edit"] = authz.Can(c, "launch.update", launch)
```

```go
type Decision struct {
    Allowed bool
    Reason  string
}

authz.Allow()
authz.Deny("only the author can edit a launch")
```

## Composing policies

```go
authz.OwnedBy[*models.User, *models.Launch](ownerID)  // compares ids, no query
authz.AnyOf(isOwner, isModerator)                     // first Allow wins
authz.AllOf(isOwner, isPublished)                     // first Deny wins
```

`OwnedBy` takes a function returning the owner's id as a string and compares
it to `auth.UserID`. It issues no query — the resource is already loaded — and
it refuses an empty identifier on either side, so an unsaved resource is never
owned by an anonymous visitor.

### Granting or denying everything at once

```go
authz.Before(r, func(c app.Context, u *models.User) (authz.Decision, bool) {
    if u.IsAdmin {
        return authz.Allow(), true   // true: this decision stands
    }
    return authz.Decision{}, false   // false: fall through to the policy
})
```

The second return value works in both directions: a `Before` can grant
everything to an administrator, or deny everything during a read-only
maintenance window.

## Authorize from the route table

When the resource comes from a route parameter, use `Owns` instead of calling
`Authorize` in the handler. The middleware's load *is* the handler's load, so
the check costs one query per request rather than two:

```go
r.Put("/launch/{slug}",
    auth.Protected,
    authz.Owns("launch.update", loadLaunchBySlug),
    handlers.UpdateLaunch,
)

func UpdateLaunch(c app.Context) error {
    launch, _ := authz.Resource[*models.Launch](c)   // already loaded
    …
}
```

A load error is returned unchanged, so a missing row becomes whatever your
repository's not-found maps to — usually a 404, which `authz` leaves alone.

`Gated` is the resourceless form, and goes after `auth.Protected`:

```go
r.Post("/launch", auth.Protected, authz.Gated("launch.create"), handlers.CreateLaunch)
```

## Failure behaviour

| Situation | Result |
|---|---|
| Ability never registered | Denied, and logged — a typo cannot open a route |
| Provider not registered | **500**, not 403: the application is misconfigured, the caller did nothing wrong |
| Resource of the wrong type | Denied, with a reason naming both types. Never a panic |
| Typed nil subject (`(*User)(nil)`) | Treated as unauthenticated → 401 |
| Same ability registered twice | **Panics at boot** — one of the two policies would never run |

## Refusals in the reader's language

The five refusals this module produces itself are translatable. Mount its
messages:

```go
&i18nprovider.Provider{
    Supported: []string{"en", "fr"},
    Mount:     map[string]fs.FS{"authz": authz.Catalogue()},
}
```

and publish them to change the wording:

```shell
lemmego run publish --tags=authz-i18n
```

which writes `resources/lang/vendor/authz/en/errors.json`. Edit the keys you
care about; the rest still come from here.

Your own `Deny` reasons are untouched. `authz.Deny("only an owner can change
a circle")` keeps saying exactly that — a reason is looked up as a key first
and used verbatim when it is not one — so you opt in per rule by returning a
key instead.

The reasons that name an ability, like `no rule is registered for "launch.update"`,
stay in English deliberately: those are programming errors, and the reader
they are addressed to is you.

## Limitations

No roles or permissions tables, no ABAC, no policy caching. A policy is a Go
function over data you have already loaded. Once a rule needs its own storage
it belongs in your application, where it can be migrated and tested like any
other feature.

## Next steps

- [Roles and permissions](/docs/rbac/) — the same decision when it has to live
  in rows rather than in Go, for a role somebody creates at runtime.
- [Authentication](/docs/security/authentication) — establishing who is asking.
- [Middleware](/docs/http/middleware) — where `Owns` and `Gated` sit in the chain.
