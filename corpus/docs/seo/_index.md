---
title: SEO
type: docs
prev: docs/search/
next: docs/payments/
weight: 133
sidebar:
  open: true
---

`lemmego/seo` gives you meta tags, JSON-LD, sitemaps, `robots.txt` and slugs.
The document head is rendered **in Go**, which is what keeps it correct when
server-side rendering fails.

```go
func Show(c app.Context) error {
    seo.Share(c, seo.Meta{
        Title:       user.Name + " (@" + user.Handle + ")",
        Description: user.Headline,
        Canonical:   "/@" + user.Handle,
        Type:        "profile",
        Image:       user.AvatarURL,
        JSONLD:      []any{personFor(user)},
    })
    return inertia.Respond(c, "Profiles/Show", props)
}
```

## Install

```shell
go get github.com/lemmego/seo
```

## Why the head is rendered in Go

{{< callout type="warning" >}}
**Metadata emitted by a page component disappears exactly when you cannot see
that it has.**

Inertia falls back to the client-rendered container on any SSR error. On that
path the head it would have produced is empty — so a `<Head>` in your React
page renders nothing, while the page still answers **200** and still looks
correct to a person. Only a crawler notices, and only weeks later.
{{< /callout >}}

Rendering the head in Go makes it independent of the Node process, so it is
correct on a cold boot, with SSR down, mid-deploy, and with Node not installed
at all. With the SSR process stopped, a page's body degrades to client
rendering while its title, canonical URL and JSON-LD all survive.

Leave your page component owning `document.title` for client-side navigation,
and nothing else.

## Register the provider

```go
&seo.Provider{
    Defaults: seo.Defaults{
        BaseURL:  config.MustEnv("APP_URL", "http://localhost:1490"),
        SiteName: "Example",
        NoIndex:  config.MustEnv("APP_ENV", "development") != "production",
    },
    Sitemap: &seo.Sitemap{
        CacheTTL: time.Hour,
        Robots: seo.Robots{
            SitemapURL: baseURL + "/sitemap.xml",
            Disallow:   []string{"/settings", "/api/"},
        },
    },
    Sources: func(a app.App) []seo.Source {
        return []seo.Source{profiles.NewSource(a), launches.NewSource(a)}
    },
}
```

One line in the Inertia root template, **above** `{{ .inertiaHead }}`:

```gohtml
{{ .seoHead }}
{{ .inertiaHead }}
```

Then register `seo.Middleware()` after the session middleware, so a handler
that forgets to call `Share` still emits a title, a canonical URL and the
right robots directive rather than nothing.

## Defaults

The zero `Meta` still renders a title, a canonical URL and a robots directive,
so a handler that sets nothing produces a page a crawler reads correctly
rather than one with no canonical and an accidental `index` in staging.

{{< callout type="warning" >}}
**Set `NoIndex: true` in every environment that is not production.** A staging
deploy indexed alongside production splits its own ranking and is tedious to
undo.

One consequence for your tests: outside production every page reports
`noindex` whatever its `Meta` asked for, so assert `Meta` values where they
are set rather than against the rendered head.
{{< /callout >}}

`Canonical` may be a path; `BaseURL` is prepended. An already-absolute value is
left alone, so an image on a CDN domain survives.

## JSON-LD

```go
seo.Meta{JSONLD: []any{
    map[string]any{
        "@context": "https://schema.org",
        "@type":    "Person",
        "name":     user.Name,
        "url":      baseURL + "/@" + user.Handle,
    },
}}
```

Each value is marshalled into its own `<script type="application/ld+json">`.

This package does not model schema.org. Pass your own structs or a map, so a
schema type it has never heard of needs no release here.

Values are escaped so they cannot break out of the script element — a
`</script>` inside a string would otherwise be an XSS hole rather than a
formatting bug.

## Sitemaps

```go
type Source interface {
    Name() string                                              // "profiles"
    Count(ctx context.Context) (int, error)
    Page(ctx context.Context, offset, limit int) ([]URL, error)
}
```

`Page` paginates rather than returning a slice, so a sitemap for a million
profiles never becomes a million-element slice. The 50,000-URL limit per file
means it never needs to.

| Path | Contents |
|---|---|
| `/sitemap.xml` | Always an **index**, even for one small source |
| `/sitemap-profiles-1.xml` | One shard of one source |
| `/robots.txt` | `Robots`, rendered |

`/sitemap.xml` is always an index so that its shape is stable: a site that
outgrows one file does not have to change the URL it already submitted to
every search engine.

An empty source serves an **empty sitemap rather than a 404**, because a
source with nothing in it yet is a normal state, and a 404 on a submitted
sitemap URL is reported as an error for as long as it lasts.

`Sources` is a function rather than a field because a source needs a
repository, a repository needs a connection, and neither exists where a
provider literal is written — `LoadProviders` runs before anything is
registered.

## Slugs

```go
seo.Slugify("Hello, World!", 60)          // "hello-world"

slug, err := seo.UniqueSlug(ctx, title, "launch", 60,
    func(ctx context.Context, s string) (bool, error) {
        return repo.SlugTaken(ctx, s)
    })
```

`Slugify` truncates on a **word boundary**, so a long title becomes a readable
slug rather than one ending mid-word.

`UniqueSlug` appends a suffix until the `taken` callback says the slug is free,
is **bounded** so a broken callback cannot loop forever, propagates a lookup
error rather than guessing, and falls back to the supplied word when a title
yields nothing.

{{< callout type="info" >}}
Make `taken` return true for **reserved** slugs as well as taken ones. A slug
is derived rather than typed, so no validator can reject it, and a reserved
word reaches the database as a constraint violation at insert time — long
after the person has finished writing.
{{< /callout >}}

## Unfurling a link: `seo/opengraph`

```go
fetcher := opengraph.New(opengraph.Options{})
preview, err := fetcher.Fetch(ctx, submittedURL)
// preview.Title, .Description, .Image, .SiteName, .Type, and .URL after redirects
```

A page with no metadata returns an empty `Preview` and no error, since that is
an ordinary page — show the URL you would have shown anyway.

### SSRF protections

Fetching a URL somebody typed means making a request from inside your network
to a destination they chose. These are the protections that make it safe, and
they are worth knowing if you ever write something similar yourself.

- **The address is checked at dial time, not before.** A hostname is not a
  destination. Resolving a name, checking the result, and then handing the
  *name* to a dialer leaves a window in which the dialer's own lookup returns
  something else — a DNS rebind. The check runs in `net.Dialer.Control`, after
  resolution and before connect.
- **Every redirect hop is checked**, because each one dials again. A public URL
  that redirects to `http://169.254.169.254` is the ordinary shape of this
  attack, and that address returns cloud credentials on AWS, GCP, Azure and
  DigitalOcean.
- **Connections are not pooled.** A reused connection was checked when it was
  first dialled, and the next fetch to that host would skip `Control` entirely.
- **`http` and `https` only**, with a byte cap, a timeout and a redirect cap.

Blocked: loopback, private, link-local, unspecified, multicast, CGNAT, NAT64,
the reserved and documentation ranges, and IPv4-mapped IPv6 forms of all of
them — `::ffff:127.0.0.1` is loopback, and every `Is…` predicate answers false
for it until it is unmapped.

It does not run JavaScript, so a page that writes its own meta tags from a
script unfurls to whatever is in the served HTML.

## Declaring the page's language

`og:locale` is emitted when it has a value. Set the site's fallback on the
provider:

```go
&seo.Provider{
    Defaults: seo.Defaults{
        BaseURL: "https://example.com",
        Locale:  "en_US",
    },
}
```

Open Graph wants `language_TERRITORY` with an underscore, not the hyphen a
language tag uses. `seo.OpenGraphLocale("en-US")` converts one for you.

If you have localized your application, you don't need to do anything else:
`Share` fills the per-page value from the request's own language, and your
default supplies the territory when it names the same language — so a reader
on `en` still gets `en_US`, and a reader on `fr` gets `fr`.

## Limitations

No Open Graph image generation, no `hreflang`, no AMP.

## Next steps

- [Inertia](/docs/frontend/inertia) — where `{{ .seoHead }}` goes in the root
  template, and how SSR failure is detected.
- [Deployment](/docs/deployment/) — setting `APP_URL` so canonical URLs are
  right in production.
