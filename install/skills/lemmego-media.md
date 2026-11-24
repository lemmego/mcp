## When to use this skill

When accepting an image upload, or generating the renditions a page serves.

Put this in front of **every** image upload. `fsys` writes submitted bytes
verbatim, which publishes the photographer's GPS coordinates; see the
`lemmego-fsys` skill.

## Orientation must be applied before metadata is stripped

Re-encoding discards EXIF, which it has to — a phone photograph carries GPS
coordinates, a device serial number and a timestamp.

But stripping EXIF is not safe on its own. A phone does not rotate the pixels
it writes: it stores the sensor's landscape frame and adds a tag asking the
viewer for a quarter turn. `image/jpeg` ignores that tag, and so does every
browser once it is gone. **So discarding metadata without first applying
orientation turns every portrait photograph on its side** — a privacy fix that
visibly breaks the feed.

The module does this in the right order already. The reason to know it: do not
"optimise" by skipping the orientation read, and do not reach for a general
EXIF library. `media/exif` reads exactly one field on purpose.

## Nothing is upscaled, so read the result

A 32-pixel avatar asked for a 256-pixel variant stays 32 pixels. Enlarging adds
bytes and no detail.

**Read the dimensions in the result rather than assuming the ones you asked
for.** `FitCover` applies the same rule to the box: a 40×40 slot given a 30×50
photograph yields 30×30 — the requested shape, at the size the source can
supply. Code that writes the requested size into a database column will be
wrong for small sources.

## The decode bomb guard

`MaxPixels` defaults to forty megapixels, checked from the **header** via
`image.DecodeConfig` before anything is allocated. A 50,000 × 50,000 PNG is a
few kilobytes of deflate that asks for ten gigabytes of RGBA, so checking after
decoding is checking after the damage.

Do not raise `MaxPixels` to accommodate one user's camera without knowing what
you are admitting; forty megapixels already covers every phone and most
cameras.

## Clean up after a failed row insert

A conversion that fails partway deletes what it already wrote, so the module
leaves nothing orphaned.

One level up is yours. A post whose row fails to insert has already had its
images stored:

    result, err := manager.Convert(...)
    // ... row insert fails ...
    manager.Remove(ctx, result.Paths()...)

`Remove` treats a file that was never written as success, because whether a
particular write landed is exactly what a rollback cannot know. Orphaned files
are invisible until somebody measures the bucket.

## Formats

The output format is chosen from the pixels rather than from the submitted
filename. **WebP decodes but does not encode** — a WebP upload is accepted and
re-encoded as something else, so do not promise WebP output. Animated GIFs keep
their animation.

`ErrNotAnImage` is what unsupported bytes return; treat it as a validation
failure, not a server error.

## Pure Go

No CGO, no external binaries, so it works in a scratch container and in CI with
nothing installed. Do not add an ImageMagick or libvips path beside it.

## Check first

`search-docs` section `media`, `module-api` on `media`.
