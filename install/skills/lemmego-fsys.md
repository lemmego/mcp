## When to use this skill

When storing or serving a file: an upload, a generated export, anything that
does not belong in the database.

## One interface, four drivers

    fs, err := fsys.Open(driver, fsys.Settings{...})

`local`, `s3`, `gcs`, `memory`, registered the way `database/sql` drivers are.
`memory` is for tests. Write against the `FS` interface and let configuration
pick the driver — code that reaches for `NewLocalStorage` directly is code that
cannot be deployed to object storage.

## Two things to get right before serving an upload

**`Upload` writes the submitted bytes verbatim.** It does not resize, re-encode
or strip metadata — so a photograph taken on a phone keeps its EXIF block, **GPS
coordinates included**, and anybody who can fetch the file can read them. Put
`lemmego/media` in front of an image upload; see the `lemmego-media` skill.

**Never derive a stored filename from what the client sent.** A submitted
filename is attacker-controlled: it can carry path separators, a second
extension, or a name your own code will later treat as code. Sniff the content
type and name the file from the format you detected.

Those two are the whole security surface of an upload path, and neither is
something the module can do for you.

## Settings are strings

`Settings` is a `map[string]string`, which is what lets a driver be chosen by
configuration. It also means a typo in a key is not a compile error — read the
driver's documented keys rather than guessing, and fail the boot if a required
one is missing instead of discovering it on the first upload.

## Signed URLs

Prefer a signed URL over proxying bytes through the application for anything
large: a proxied download occupies a request goroutine for its whole duration,
and a slow client then costs you a worker rather than a redirect.

## Writing your own driver

`fsys.Register(driver, factory)` with a `Factory func(Settings) (FS, error)`.
Register from an `init` in your own package and blank-import it, the same shape
as a SQL driver.

## Check first

`search-docs` section `filesystem`, `module-api` on `fsys`.
