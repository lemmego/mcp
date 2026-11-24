---
title: Consent screen
type: docs
prev: docs/oauth2/protecting-routes
next: docs/oauth2/tokens-and-keys
weight: 146
sidebar:
  open: true
---

## The default

Out of the box the consent screen is a single self-contained HTML document:
inline styles, no JavaScript, no external assets, and it follows the visitor's
light or dark preference.

Self-containment is the point. A project rendering with Templ or Inertia has
no Go template cache for this route and no page component for it, so anything
depending on either would work in one project and fail in the next. Nothing
needs publishing or configuring for the flow to work.

## Replacing it

```go
&oauth2.Provider{
    ConsentView: func(w io.Writer, data oauth2.ConsentData) error {
        return myTemplates.ExecuteTemplate(w, "consent.html", data)
    },
}
```

`ConsentData` carries everything the screen needs:

| Field | |
|---|---|
| `ClientName`, `ClientID` | The application asking |
| `Scopes` | `[]ConsentScope` of `Name` and `Description` |
| `RedirectURI`, `RedirectHost` | Where the user will be sent |
| `FormAction` | Where to post the decision |
| `CSRFToken` | The framework's token, when one exists |
| `Nonce` | This module's own token — **always include it** |
| `Untrusted` | True unless the client is first-party |

The form must post `decision=approve` or `decision=deny`, and must include
`_oauth_nonce`.

### With Templ

```go
ConsentView: func(w io.Writer, data oauth2.ConsentData) error {
    return views.Consent(data).Render(context.Background(), w)
},
```

### With Inertia

An Inertia page is a component in a JavaScript bundle this module knows
nothing about, so an Inertia project **must** override the view — the default
would render plain HTML in the middle of a single-page application.

```go
ConsentView: func(w io.Writer, data oauth2.ConsentData) error {
    // Render your own page component, or write an HTML document that boots
    // the relevant part of your bundle.
}
```

## Why the decision is rebuilt from the session

When the screen is rendered, the authorization is stored in the session. When
the decision arrives, the grant is built from **that**, not from the posted
form.

Without it, a page that displayed "Read your orders" could post back
`scope=orders:read orders:write admin` and the server would have no way to
know the user never saw it. What gets granted is what was shown.

The pending authorization is single use and expires after fifteen minutes.

## Two anti-forgery tokens

The form carries the framework's `_token` when it exists, and always carries
this module's `_oauth_nonce`.

The second is not redundant. The REST API preset installs no CSRF middleware
at all, so `_token` would be empty there — and a consent form with no
forgery protection is a form an attacker can submit on a user's behalf. The
nonce is bound to the session and to the specific request being shown, so one
lifted from another user's page is useless.

## Identifying the user

By default the module asks `auth` for the id of whoever is signed in
(`auth.UserID`). That is one value on every path, because a credential carries
an id and nothing else — there is no shape to unpack. Consent only needs the
id, so no user row is read here.

A visitor who is not signed in is redirected to `login_route`, not refused
with a 401 — this is a browser flow, and a person who is not signed in should
be given the chance to sign in.

To identify users another way:

```go
&oauth2.Provider{
    ResourceOwner: func(c app.Context) (string, bool) {
        id, ok := c.Get("current_user_id").(string)
        return id, ok
    },
}
```

Return `false` to send them to the login route.

## Skipping consent

A first-party client — one your own application owns — skips the screen
entirely, since there is nobody to ask:

```bash
lemmego run oauth:client --first-party --name "Acme Web" --redirect-uri ...
```

Set `skip_consent_for_first_party` to `false` to show it anyway.

## The device screens

`DeviceView` replaces the code-entry and confirmation screens the same way,
receiving a `DeviceData`.

## Next steps

- [Tokens and keys](/docs/oauth2/tokens-and-keys) — what approval actually
  issues.
- [Clients](/docs/oauth2/clients) — the first-party clients that skip it.
