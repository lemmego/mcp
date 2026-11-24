## When to use this skill

When adding a translated message, a plural, or a locale.

## The file layout is the key

    resources/lang/{locale}/{group}.json

So `en/validation.json` holding `{"required": "…"}` yields the key
`validation.required`. There is no key registry to update — the path is the
key.

## Plurals use CLDR form names, not Laravel's pipe syntax

    {
      "apples": { "one": "{{.Count}} apple", "other": "{{.Count}} apples" }
    }

**Not** `"apple|apples"`. This is the one place the module deliberately
diverges from Laravel, and the reason is a real bug: Laravel's selector is a
pre-CLDR switch, and its `choose` returns the *singular* when a language has
more forms than the translator wrote. Languages with three to six plural forms
make that a wrong answer rather than an ugly one.

A consequence worth knowing: **a CLDR form name is reserved everywhere**,
including as an ordinary nested key. An object that mixes form names with other
keys is a **load error**, not a warning, because a JSON object is genuinely
ambiguous between a plural set and a group.

## A module ships its own messages under a namespace

`authz::errors.forbidden` — the `::` separates the namespace. An application
overrides one message by publishing:

    resources/lang/vendor/<module>/<locale>/

Do not edit the module's own catalogue.

## Go does not format numbers, currency or dates here

This is deliberate and permanent, not a gap to fill. `x/text/message` does not
format `time.Time` at all, so a "Go formats everything" story would have been
false on the day it shipped. Formatting happens **in the browser through
`Intl`**, with the locale arriving as a page prop.

If you are tempted to add a Go-side formatter: the only surface that justifies
one is something with no browser — a PDF, a mail body — and that is the
argument to make when such a surface exists. Until then, pass the locale and
format client-side.

## Locale resolution needs no redirects

`i18n/provider` negotiates each request's locale from `?lang=`, a cookie, a
stored preference and `Accept-Language`. **Locale-prefixed URLs (`/fr/...`) and
`hreflang` alternates are not implemented**, and that is a considered position:
a prefix needs every redirect in the application to re-prefix itself, and a
`Location` rewriter that mishandles a cross-origin target is an open redirect.

`i18n/i18ninertia` shares the catalogue with the browser as a `once` prop, so
the same message set is available both sides. `api/app` declares a two-method
`Translator` interface, so the framework core depends on none of this.

## Check first

`search-docs` section `localization`, `module-api` on `i18n`.
