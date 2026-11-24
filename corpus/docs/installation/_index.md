---
title: Installation
type: docs
prev: docs/glossary
next: docs/quick-start/
weight: 12
sidebar:
  open: true
---

You'll be installing one thing: the `lemmego` command. It is the tool you use
to create a project, run it while you work on it, generate files and apply
database changes — so everything else in these docs starts with it being on
your PATH.

## Prerequisites

You'll want to be comfortable with Go. Beyond that:

- **[Go 1.27 or newer](https://go.dev/dl/)** — required, not optional. The ORM
  uses generic methods, which only exist from 1.27.
- **[Node.js](https://nodejs.org/en/download/package-manager)** — only if you
  are building a frontend with JavaScript or CSS. Lemmego bundles those with
  Vite, which runs on Node. A Go-templates-only project does not need it.
- **[Air](https://github.com/air-verse/air)** — watches your `.go` files and
  rebuilds and restarts your application when you save one, so you are not
  stopping and starting it by hand. Only needed for `lemmego dev`.
- **[Templ](https://templ.guide/quick-start/installation)** — optional. A
  template language that compiles your templates into Go code, so a typo in a
  field name is a build error rather than a blank space on the page. Skip it
  unless you choose a Templ frontend.

## Installing the CLI

With those in place, install the CLI:

{{< tabs >}}
{{< tab name="Linux" >}}

```shell
curl -fsSL https://get.lemmego.org | sudo sh
```

{{< /tab >}}
{{< tab name="macOS" >}}

```shell
curl -fsSL https://get.lemmego.org | sudo sh
```

{{< /tab >}}
{{< tab name="Windows" >}}

In an elevated PowerShell prompt:

```powershell
irm https://get.lemmego.org/ps1 | iex
```

{{< /tab >}}
{{< /tabs >}}

### Install from source

If you have Go installed, you can build the CLI yourself:

```shell
go install github.com/lemmego/cli/cmd@latest
```

{{< callout type="warning" >}}
This installs a binary called **`cmd`**, after the package's directory name.
Rename it to `lemmego` before using the rest of these docs:

```shell
mv "$(go env GOPATH)/bin/cmd" "$(go env GOPATH)/bin/lemmego"
```
{{< /callout >}}

Make sure `$GOPATH/bin` is in your `PATH`.

A binary installed this way reports its module version; one built from a
checkout reports `dev+<commit>`.

### Installer options

Both installers resolve the latest release automatically and verify the download
checksum. Two environment variables override that:

| Variable | Effect |
|----------|--------|
| `LEMMEGO_VERSION` | Install a specific tag instead of the latest, e.g. `v0.4.1` |
| `LEMMEGO_INSTALL_DIR` | Install somewhere other than `/usr/local/bin` (`C:\Windows\System32` on Windows) |

```shell
# Pin a version, and install without sudo
LEMMEGO_VERSION=v0.4.1 LEMMEGO_INSTALL_DIR=$HOME/.local/bin \
  curl -fsSL https://get.lemmego.org | sh
```

## Verify Installation

Run the following to confirm the CLI is installed:

```shell
lemmego --version
```

Expected output:

```
version [x.x.x]
```

## Next steps

Now that the CLI is installed, proceed to the [Quick Start](/docs/quick-start/) guide to create your first project.
