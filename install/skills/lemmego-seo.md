## When to use this skill

When a page needs a title, a description, Open Graph tags, JSON-LD, a sitemap
or a slug.

## Render the head in Go, not in the page component

**Metadata emitted by a page component disappears exactly when you cannot see
that it has.**

Inertia falls back to the client-rendered container on any SSR error. On that
path the head it would have produced is empty — so a `<Head>` in a React page
renders nothing, while the page still answers **200** and still looks correct
to a person. Only a crawler notices, and only weeks later.

Rendering the head in Go makes it independent of the Node process: correct on a
cold boot, with SSR down, during a deploy, and with Node not installed at all.

    seo.Share(c, seo.Meta{
        Title:       post.Title,
        Description: post.Excerpt,
        Canonical:   "/posts/" + post.Slug,
    })

Leave the page component owning `document.title` for client-side navigation,
and nothing else.

This is why `seo.Middleware()` and the `seoHead` template key exist. If you
find yourself adding a `<Head>` with the same values, delete one of them — and
delete the client-side one.

## `Defaults` fills what a page leaves empty

Set the site name, the fallback description and the default image once, and a
page supplies only what differs. A page that sets nothing still emits a
complete head rather than a partial one.

## JSON-LD without a schema.org model

    seo.JSONLD(map[string]any{
        "@context": "https://schema.org",
        "@type":    "Person",
        "name":     user.Name,
    })

It returns `template.HTML` and escapes for a `<script>` context, which is not
the same as HTML escaping — a `</script>` inside a string value would
otherwise end the block. Do not build the JSON yourself with `fmt.Sprintf`.

There are no generated schema.org structs, and that is the intended shape: the
vocabulary is enormous, mostly optional, and changes without warning.

## Sitemaps do not load the table into memory

Stream them. `NewPages(lastMod, paths...)` covers the static set; a large
dynamic set is produced by iterating rather than by building one slice, because
a sitemap of a million URLs is a million rows and the naive version is an
out-of-memory error on a cron job nobody watches.

## Slugs

    slug := seo.Slugify(title, 60)
    slug, err := seo.UniqueSlug(ctx, title, fallback, 60, exists)

`UniqueSlug` takes an existence check, so uniqueness is decided against your
table rather than guessed. `ErrNoSlug` means neither the title nor the fallback
yields anything usable — a title of only punctuation or only emoji — so handle
it rather than storing an empty string.

Store the slug. Re-deriving it on read means a renamed title silently breaks
every existing link.

## Limits

No locale-prefixed URLs (`/fr/...`) and no `hreflang` alternates. `i18n`
resolves a locale without redirects; a URL prefix needs every redirect in the
application to re-prefix itself, and a `Location` rewriter that mishandles a
cross-origin target is an open redirect.

`seo/opengraph` reads **somebody else's** metadata, for link previews. Do not
confuse it with emitting your own.

## Check first

`search-docs` section `seo`, `module-api` on `seo`.
