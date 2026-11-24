---
title: File System
type: docs
prev: docs/broadcast/
next: docs/media/
weight: 130
sidebar:
  open: true
---

Your application needs somewhere to put uploaded files. On your laptop that is
a folder; in production it is usually S3 or Google Cloud Storage, because a
container's own disk does not survive a deploy.

You do not want that difference spread through your code. So you write against
a **disk** — a named place to put files — and configure what each one actually
is. `Put` and `Get` work the same either way, and moving from local storage to
S3 is a configuration change rather than a rewrite:

```go
disk, err := a.FileSystem().Disk("public")
if err != nil {
    return err
}
err = disk.Write("avatars/42.jpg", data)
```

A project starts with a `local` disk pointed at `storage/`, and you add others
as you need them.

## Architecture

```
Application Code
      |
      v
+------------------+
|  fs.FileSystem   |  <-- Multi-disk manager (api/fs)
+------------------+
      |
      v
+------------------+
|  fsys.FS         |  <-- Unified interface
+------------------+
      |
      +-- LocalStorage   (local filesystem)
      +-- S3Storage      (Amazon S3)
      +-- GCSStorage     (Google Cloud Storage)
      +-- MemoryStorage  (in-memory, for testing)
```

## FS Interface

All providers implement a common interface:

```go
type FS interface {
    Driver() string
    Read(path string) (io.ReadCloser, error)
    Write(path string, contents []byte) error
    Delete(path string) error
    Exists(path string) (bool, error)
    Rename(oldPath, newPath string) error
    Copy(sourcePath, destinationPath string) error
    CreateDirectory(path string) error
    GetUrl(path string) (string, error)
    Open(path string) (*os.File, error)
    Upload(file multipart.File, header *multipart.FileHeader, dir string) (*os.File, error)
}
```

## Framework Integration

### Provider

The filesystem is registered via the `fs.Provider` in `bootstrap/providers.go`:

```go
func LoadProviders() []app.Provider {
    return []app.Provider{
        &fs.Provider{},
    }
}
```

### Access

```go
filesys := app.Get[*fs.FileSystem](a)

// Get a specific disk
disk, err := filesys.Disk("s3")
disk, err := filesys.Disk() // Returns default disk (from FILESYSTEM_DISK env)
```

## Usage

```go
disk, _ := filesys.Disk("local")

// Write
disk.Write("uploads/report.pdf", data)

// Read
reader, _ := disk.Read("uploads/report.pdf")
defer reader.Close()

// Check existence
exists, _ := disk.Exists("uploads/report.pdf")

// Delete
disk.Delete("uploads/report.pdf")

// Get URL
url, _ := disk.GetUrl("uploads/report.pdf")

// Upload from request
disk.Upload(file, header, "avatars")
```

## Configuration

Configuration is set in `internal/configs/filesystems.go`:

```go
config.Set("filesystems", config.M{
    "default": config.MustEnv("FILESYSTEM_DISK", "local"),
    "disks": config.M{
        "local": config.M{
            "driver": "local",
            "root":   "storage/app",
        },
        "s3": config.M{
            "driver": "s3",
            "key":    config.MustEnv("AWS_ACCESS_KEY_ID", ""),
            "secret": config.MustEnv("AWS_SECRET_ACCESS_KEY", ""),
            "region": config.MustEnv("AWS_DEFAULT_REGION", "us-east-1"),
            "bucket": config.MustEnv("AWS_BUCKET", ""),
        },
        "r2": config.M{
            "driver": "s3",
            "key":    config.MustEnv("R2_ACCESS_KEY_ID", ""),
            "secret": config.MustEnv("R2_SECRET_ACCESS_KEY", ""),
            "region": "auto",
            "bucket": config.MustEnv("R2_BUCKET", ""),
            "endpoint": config.MustEnv("R2_ENDPOINT", ""),
        },
    },
})
```

## A cloud disk needs one blank import

Configuring the disk is not quite enough. A driver registers itself from an
`init` in its own package, the way a `database/sql` driver does, so the one
that serves your disk has to be imported somewhere:

```go
// bootstrap/providers.go
import (
    _ "github.com/lemmego/fsys/s3"  // or fsys/gcs
)
```

`lemmego new --disk s3` writes that line for you. Adding an S3 disk by hand to
a project that was generated with local storage does not, and the failure waits
until the disk is first opened rather than appearing at compile time:

```text
fsys: no driver "s3" is registered (have: [local memory]). A driver registers
itself when its package is imported, so this usually means a missing blank
import such as _ "github.com/lemmego/fsys/s3"
```

The arrangement exists because the alternative was worse. Serving S3 and GCS
from the root package meant every application — and `api/fs` with it — linked
both cloud SDKs whether or not it had ever seen a bucket. Measured on a trivial
program, that was the difference between **58 packages and 691**. Local and
memory are in the root and cost nothing, so they need no import.

## Providers

### Local Storage

Stores files on the local filesystem. Configured with a root directory.

```go
storage := fsys.NewLocalStorage("./storage/app")
```

Features: Read, Write, Delete, Exists, Rename, Copy, CreateDirectory, Open, Upload.

### S3 Storage

Stores files on Amazon S3, or an S3-compatible service such as Cloudflare R2.
It lives in `fsys/s3`, for the reason above:

```go
import "github.com/lemmego/fsys/s3"

storage, err := s3.New(bucket, region, key, secret, endpoint)
```

`endpoint` is empty for Amazon's own S3 and set for a compatible service. URLs
are generated as pre-signed URLs with 15-minute expiry by default.

### GCS Storage

Stores files on Google Cloud Storage, from `fsys/gcs`:

```go
import "github.com/lemmego/fsys/gcs"

storage, err := gcs.New(projectID, bucket, serviceAccountKey)
```

URLs use the standard `https://storage.googleapis.com/<bucket>/<path>` format.

Most applications never call either constructor directly: `a.FileSystem().Disk()`
builds the disk from configuration, and these are what it builds.

### Memory Storage

In-memory storage for testing purposes.

```go
storage := fsys.NewMemoryStorage()
```

All operations work in memory. `GetUrl` returns mock `mem://` URLs. `Open` is not supported.

## Next steps

- [Media & images](/docs/media/) — resizing and re-encoding before you store.
- [Configuration](/docs/configuration) — declaring your disks.
- [Deployment](/docs/deployment/) — pointing a disk at cloud storage.
