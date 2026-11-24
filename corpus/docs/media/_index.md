---
title: Media & Images
type: docs
prev: docs/filesystem/
next: docs/search/
weight: 131
sidebar:
  open: true
---

`lemmego/media` turns an uploaded image into the renditions a page actually
serves: a cleaned original, a set of sized variants, and their real dimensions
reported back so markup can reserve the space before the bytes arrive.

```go
manager, err := media.New(media.Options{Store: app.FileSystem().Disk("public")})

result, err := manager.Ingest(ctx, upload, media.Request{
    Dir:  "posts",
    Base: "9-1760000000-0",
    Conversions: []media.Conversion{
        {Name: "sm", Width: 400, Height: 400},
        {Name: "md", Width: 1000, Height: 1000},
    },
})

// posts/9-1760000000-0.jpg      the original, re-encoded, no metadata
// posts/9-1760000000-0_sm.jpg   400×267
// posts/9-1760000000-0_md.jpg   1000×667
```

## Install

```shell
go get github.com/lemmego/media
```

It is pure Go: no CGO, no subprocess, nothing to install on the host. Tools
that shell out to `jpegoptim`, `pngquant` or `svgo` leave a single-binary
deployment either carrying four host packages or silently optimising nothing,
and a silent no-op looks exactly like success.

## How optimisation works

Optimising here means **re-encoding at a chosen quality and discarding every
metadata block**, which is most of the available win: a camera JPEG at quality 95 with its EXIF and embedded thumbnail stripped
is routinely half the size at quality 82 with no visible difference.

## Metadata and orientation

Re-encoding discards EXIF, which it has to: a photograph taken on a phone
carries **GPS coordinates**, a device serial number and a timestamp, and
serving the submitted bytes back unchanged publishes all three to everybody
who can see the page.

{{< callout type="warning" >}}
Stripping EXIF is not safe on its own. A phone does not rotate the pixels it
writes — it stores the sensor's landscape frame and adds a tag asking the
viewer for a quarter turn. `image/jpeg` ignores that tag, and so does every
browser once it is gone.

**So discarding metadata without first applying orientation turns every
portrait photograph on its side**: a privacy fix that visibly breaks the feed.
{{< /callout >}}

So `media/exif` reads exactly one field, the pixels are rotated to match, and
re-encoding then discards the rest. It is not a general EXIF library.

## Sizing

**Nothing is upscaled.** A 32-pixel avatar asked for a 256-pixel variant stays
32 pixels, because enlarging adds bytes and no detail.

**So read the dimensions from the result rather than assuming the ones you
asked for.** `FitCover` applies the same rule to the box: a
40×40 avatar slot given a 30×50 photograph yields 30×30 — the requested shape,
at the size the source can supply.

## Output format

`FormatAuto` writes PNG when the source has a pixel that is genuinely
translucent, and JPEG when it does not.

This matters because most PNGs on the web are RGBA with every pixel opaque — a screenshot saved by a tool that always writes an alpha
channel. Choosing by colour model keeps those as PNG, several times larger
than the JPEG they could be.

It is re-decided per variant, not inherited from the source: scaling
resamples, so an edge that was opaque can become translucent and a lone
translucent pixel can be lost to a downscale.

Asking for `FormatJPEG` from a transparent source is allowed, and the image is
composited **onto white** first. JPEG has no alpha channel and the encoder
reads the colour values as given — so a transparent pixel, stored as
transparent *black*, encodes as black, and a logo arrives on a black
rectangle.

### WebP input

`x/image` ships no WebP encoder, and every pure-Go one is either incomplete or
a CGO binding. A WebP upload is accepted and its renditions are written as
JPEG or PNG.

### Animated GIFs

Re-encoding one writes its poster frame and throws the animation away —
silently, because the result is a valid image of the right size. So the
submitted bytes are preserved for an animated GIF whatever `Original` says,
and `Result.OriginalRaw` reports it.

Detection walks the GIF block structure rather than decoding. The only
standard-library function that reports a frame count is `gif.DecodeAll`, which
holds **every frame in memory at once**, so a few megabytes of LZW — well
inside any sane upload limit — expands to gigabytes of `*image.Paletted`.

## Input size limits

`image.DecodeConfig` parses only the header, so the pixel count is known
before anything is allocated — and a 50,000 × 50,000 PNG is a few kilobytes of
deflate that asks for ten gigabytes of RGBA. Checking the size after decoding
is checking it after the damage.

`MaxPixels` defaults to forty megapixels, which covers every phone and most
cameras. The guard multiplies as `int64`, because two 32-bit dimensions
multiply past `MaxInt32` and a narrower guard would overflow into a small
positive number — admitting the one image it exists to refuse.

## Cleanup on failure

A conversion that fails after two variants were written deletes those two
before returning the error, so a failed ingest leaves nothing behind.

One level up it is your job, and `Result.Paths()` is for it: a post
whose row fails to insert has already had its images stored, so hand those
paths to `Manager.Remove`, which treats a file that was never written as
success.

## The `Store` interface

```go
type Store interface {
    Write(path string, contents []byte) error
    Read(path string) (io.ReadCloser, error)
    Delete(path string) error
    Exists(path string) (bool, error)
}
```

[`lemmego/fsys.FS`](/docs/filesystem/) satisfies this **structurally**, so you
pass the handle you already have and this package never imports `fsys`. That
keeps the dependency out of your build: `fsys` serves S3 and GCS from one
package, so importing it would pull in the AWS and Google Cloud SDKs.

The root package builds `golang.org/x/image` and nothing else.
`media/provider`, which needs the container, reaches 147 modules.

## Paths

`dir/base.ext` for the original and `dir/base_name.ext` for a variant, so a
caller that pinned the format can build a URL without reading a row back —
`VariantPath` is that derivation.

It is unpredictable for exactly one case and says so with a second return
value: `FormatAuto` chooses between PNG and JPEG by scanning the source, so the
extension is not known until the pixels have been read. Pin a format on any
conversion whose URL you intend to construct rather than store.

## Register the provider

```go
// bootstrap/providers.go
&mediaprovider.Provider{Disk: "public"},
```

`MEDIA_DISK`, `MEDIA_MAX_PIXELS`, `MEDIA_MAX_BYTES` and `MEDIA_QUALITY`
override it. An empty value means "use the default" rather than being taken as
a decision, and an unparsable one is reported rather than panicking.

A disk that cannot be resolved **fails the boot**, rather than letting an
application serve upload forms and discover at the first upload that it has
nowhere to put the file.

## Testing

`media/mediatest` has an in-memory store, a contract suite any `Store` must
pass, and the fixtures that make the interesting cases testable at all: a JPEG
carrying an orientation tag, a PNG whose alpha channel is present but unused,
an animated GIF, and a decode bomb.

{{< callout type="info" >}}
If you write your own decode-bomb fixture: `image/png` verifies the IHDR
checksum in `DecodeConfig`, so a rewritten width without a recomputed CRC is
rejected as corrupt and never reaches the pixel guard. The guard can only be
tested with a well-formed header that lies.
{{< /callout >}}

## Limitations

**WebP and AVIF output** — no pure-Go encoder for either is ready ·
**video** · **presigned uploads** — `fsys` has no presigned PUT ·
**deferred conversion** · **a focal point for cropping** — `FitCover` centres,
and face detection is not available as a pure-Go dependency.

## Next steps

- [Filesystem](/docs/filesystem/) — the disks a `Store` is built from.
- [File uploads](/docs/http/request-handling#file-uploads) — getting the upload
  that `Ingest` takes.
