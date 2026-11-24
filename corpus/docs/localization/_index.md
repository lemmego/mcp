---
title: Localization
type: docs
prev: docs/payments/
next: docs/oauth2/
weight: 135
sidebar:
  open: true
---

Your application has text in it: button labels, validation messages, the
sentence on a confirmation email. Right now every one of those is an English
string typed into the Go file or the React component that uses it, which is
fine until somebody wants the site in a second language — and then it is a
search through every file you have.

`lemmego/i18n` moves those strings into JSON files, one directory per
language, and gives you one function to look them up by name. Your code asks
for `validation.required` and gets back whatever the reader's language says
that is.

It handles **translation**: messages, plurals, placeholders, and deciding
which language a request is in. It does not format numbers, dates or
currencies — see [Limitations](#limitations), where you'll find a shorter and
better answer for those.

## Install

```bash
go get github.com/lemmego/i18n
```

Register the provider in `bootstrap/providers.go`, **before** your Inertia
provider:

```go
&i18nprovider.Provider{
    Supported: []string{"en", "fr"},
}
```

The first locale is both the default and the fallback. Create the catalogue:

```
resources/lang/
  en/validation.json
  en/pages.json
  fr/pages.json
```

That is all the setup there is. A request now resolves to a language and
`ctx.T()` returns a translator for it.

## Your first message

The key comes from the file layout. `resources/lang/en/pages.json`:

```json
{
  "greeting": "Hello, :name",
  "title": "Verified revenue"
}
```

gives you `pages.greeting` and `pages.title`. Nest an object to go deeper —
`{"min": {"string": "…"}}` in `validation.json` is `validation.min.string`.

In a handler:

```go
func Show(c app.Context) error {
    return c.JSON(200, app.M{
        "greeting": c.T().Get("pages.greeting", map[string]any{"name": "Ada"}),
    })
}
```

In a Go template, `t` and `choice` are already registered:

```gotemplate
{{ t "pages.greeting" "name" .User.Name }}
```

Don't worry about a key you have not written yet — it comes back as itself.
`pages.greting` renders as `pages.greting`, which is obviously wrong, easy to
spot and quick to grep for, and your page still renders. Each missing key is
counted once per process, and `Catalog.Misses()` tells you which.

## Placeholders

A `:name` in a message is replaced by the value you pass:

```json
{ "welcome": "Welcome back, :name" }
```

```go
c.T().Get("pages.welcome", map[string]any{"name": "Ada"})
```

A placeholder you don't supply is left exactly as written, so `:nmae` in the
page is a typo you can see rather than a gap you cannot.

If you have two placeholders where one name is a prefix of the other —
`:value` and `:values` — you don't need to do anything. The longer one wins,
which is the behaviour you'd expect and not the behaviour you get from a naive
replace.

## Plurals

A plural message is an object keyed by **form name**:

```json
{
  "posts": {
    "one": ":count post",
    "other": ":count posts"
  }
}
```

```go
c.T().Choice("pages.posts", 3)   // "3 posts" — :count is bound for you
```

The form names are CLDR's: `zero`, `one`, `two`, `few`, `many`, `other`. You
only write the ones your language uses — English needs two, French needs two
but counts 0 as `one`, Russian needs three, Arabic needs all six. Which ones a
language needs is not something you have to know: name them and the right one
is chosen.

`other` is required, and it is what you get if a form is missing. That matters
more than it sounds: it means an incomplete translation renders a plural, never
a singular with a count of five next to it.

For an exact count, use `=N`, which is checked before the form:

```json
{
  "posts": {
    "=0": "nothing here yet",
    "one": ":count post",
    "other": ":count posts"
  }
}
```

### A form name is a reserved word

Because an object of form names *is* a plural message, you cannot also use
`one` or `other` as an ordinary nested key. An object that mixes the two is a
load error naming the file and the key, rather than a guess — so if you see
that error, flatten the group:

```json
// not this
{ "active_url": { "other": "…", "status": "…" } }

// this
{ "active_url": "…", "active_url_status": "…" }
```

## Which language a request is in

Resolved for you, in this order, first match winning:

| Source | Enabled by |
|---|---|
| `provider.Prefer(c, "fr")` from a handler | always |
| `?lang=fr` | setting `QueryKey: "lang"` |
| the reader's stored preference | setting `UserLocale` |
| a `locale` cookie | always |
| `Accept-Language` | always |
| the first supported locale | always |

`?lang=` is off unless you ask for it, because it is a switch anybody can flip
on anybody else's link. When you do turn it on, the chosen language is written
to the cookie for you — you need that, because validation errors are rendered
in the language of the form post and read back after a redirect.

A reader's stored preference is read **lazily**, on the request's first
translation. You don't have to arrange that: the negotiating middleware runs
before your route's own authentication handler, so there is nobody to ask yet
when it runs, and asking later is what makes the answer right.

```go
&i18nprovider.Provider{
    Supported: []string{"en", "fr"},
    QueryKey:  "lang",
    UserLocale: func(c app.Context) (string, bool) {
        user, ok := auth.UserAs[*models.User](c)
        if !ok || user.Locale == "" {
            return "", false
        }
        return user.Locale, true
    },
}
```

Return `false` for "this person has not chosen", which is different from
choosing the default and is treated differently.

The language is settled by the first translation on a request and does not
change after that. `provider.Prefer` returns `i18n.ErrLocaleAlreadyUsed` if
you call it too late, rather than doing nothing — otherwise you could render
the top of a page in one language and the bottom in another.

## Validation messages

These come free. The framework ships its own English for every built-in rule,
so a validator says sentences with no catalogue at all. To change one, publish
the files and edit them:

```bash
lemmego run publish --tags=i18n
```

That writes `resources/lang/en/validation.json` and friends, and refuses to
overwrite a file you have already edited.

The built-in messages say "This field is required" rather than naming the
field, because both the Inertia and template error paths render a message next
to its own input. If you want the Laravel voice, the `:attribute` placeholder
is bound on every message anyway — publish the file and use it:

```json
{ "required": "The :attribute field is required." }
```

`:attribute` resolves from `Validator.Attributes(...)`, then
`validation.attributes.<field>`, then a humanised field name, so it is never
empty.

## Reaching the browser

With Inertia, add one more provider **after** your Inertia one:

```go
&i18ninertia.Provider{}
```

Your pages now receive an `i18n` prop holding the active language and its
whole message set. Add the language to the root template:

```html
<html lang="{{ .htmlLang.Lang }}" dir="{{ .htmlLang.Dir }}">
```

`lang` and `dir` are rendered in Go rather than by a page component on
purpose. If the server-side renderer fails, Inertia falls back to
client-rendering and the head it would have produced comes back empty — so
anything the component owns disappears exactly when you cannot see it
happening. Rendered in Go, the attributes survive.

On the client, vendor `i18n.mjs` and `i18n.d.mts` from the module's `js/`
directory into `resources/js/lib/`, wrap each page in a provider, and use the
hook:

```tsx
const { t, choice } = useT();

return <h1>{t("pages.title")}</h1>;
```

`t` and `choice` follow the same two rules as Go — longest placeholder first,
and the form name chosen from the same CLDR data — and a shared fixture in the
module is read by a Go test and a `node --test` so the two cannot drift.

The catalogue is sent **once** per page session, not on every visit. If you
use partial reloads (`router.reload({ only: [...] })`), the prop is absent from
those responses, so hold the last one you saw — the provider shipped with the
module does this, and it is the reason it does.

## Translating email

A mail body goes in a subdirectory named after the language:

```
resources/mail/
  verify_email.gohtml
  fr/verify_email.gohtml
```

Wire the two sides together once:

```go
&mail.Provider{Locale: i18n.LocaleOf}
```

and, for notifications, tell the mail channel how to carry a recipient's
language:

```go
mailchan.New(mailer,
    mailchan.WithLocale(func(ctx context.Context, locale string) context.Context {
        return i18n.With(ctx, localizer.For(locale))
    }),
)
```

A recipient says which language it wants by implementing one method:

```go
func (u *User) PreferredLocale() (string, bool) {
    return u.Locale, u.Locale != ""
}
```

A template you have not translated falls back on its own, not as a whole
language — so `fr/verify_email.gohtml` can exist while `fr/receipt.gohtml`
does not, and each arrives in the best language available. A hand-written
`.gotxt` text part in the base directory is still used.

The language is resolved when you call `Send` or `Queue`, before anything is
enqueued. You never have to think about a worker having a locale, because it
never needs one: what gets queued is the finished message.

## Messages from a module

A module ships its own messages under a namespace. Mount it once:

```go
&i18nprovider.Provider{
    Supported: []string{"en", "fr"},
    Mount:     map[string]fs.FS{"authz": authz.Catalogue()},
}
```

and its keys resolve as `authz::errors.forbidden`. To change one, publish it
and edit the copy:

```bash
lemmego run publish --tags=authz-i18n
```

which writes `resources/lang/vendor/authz/en/errors.json`. Your copy wins for
the keys it names; anything it leaves out still comes from the module, so a
partial override stays partial.

## Checking a translation is complete

```go
missing := catalogue.Diff("en", "fr")
```

returns the keys the default locale has and `fr` does not. Whether that fails
your build is your call — put it in a test if it should.

A malformed catalogue fails at boot, with the file and the position in it:
bad JSON, a non-string message, an unknown plural form, a mixed plural object,
or a directory whose name is not a canonical language tag. That last one is
worth knowing about: `resources/lang/pt-br/` is rejected and names `pt-BR` as
what you meant, because the difference is invisible on macOS and fatal in a
Linux container.

## Limitations

**No number, currency or date formatting.** Use `Intl` in the browser, with
the locale from the `i18n` prop:

```ts
new Intl.DateTimeFormat(locale, { month: "short", day: "numeric" }).format(d);
```

It is already correct for every language and costs nothing. Go's own
`x/text/message` does not format `time.Time` at all, so a Go-side formatter
would be half an answer.

**No locale-prefixed URLs** (`/fr/about`) and no `hreflang` alternates.
Language comes from `?lang=`, a cookie or `Accept-Language`, none of which
needs a redirect.

**No string extraction tool.** Nothing scans your components for untranslated
text. What you get instead is the miss recorder: visit a page in a second
language and the keys it is missing are counted.

**`dir` is emitted, but no right-to-left styling is included.** The attribute
is set correctly from the language's script; making a layout work in both
directions is CSS work in your application.

## Next steps

- [Validation](../http/validation/) — the rules whose messages these are
- [Mail](../mail/) — per-language email bodies
- [Inertia](../frontend/inertia/) — where the `i18n` prop arrives
