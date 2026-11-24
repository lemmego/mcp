---
title: Vite Integration
type: docs
prev: docs/frontend/inertia
next: docs/orm/
weight: 44
sidebar:
  open: true
---

Browsers cannot read your TypeScript, and they should not download forty
separate files. So something has to turn your source into a few small files a
browser can use — compiling TypeScript to JavaScript, resolving imports,
bundling it together and shrinking it. That job is called **building your
assets**, and Lemmego uses [Vite](https://vitejs.dev/) to do it.

Vite also runs while you work, and that is the part you will notice. Save a
component and the change appears in the browser immediately, without a
refresh and without losing what you had typed into the form. That is **hot
module replacement**: Vite swaps just the module you changed into the running
page.

You get two commands. `lemmego dev` runs Vite alongside your Go application
while you work; `lemmego build` produces the optimised files you deploy.

## Configuration

The generated project includes a `vite.config.js`:

```js
import { defineConfig } from 'vite';
import lemmego from '@lemmego/vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [
    lemmego({
      input: ['resources/js/app.tsx'],
      refresh: true,
    }),
    react(),
    tailwindcss(),
  ],
});
```

### Entry Points

The main entry point is `resources/js/app.tsx` (React) or `resources/js/app.js` (Vue).

The Lemmego plugin also preserves the paths consumed by the Go integration:

- `public/hot` contains the development server URL.
- `public/build/manifest.json` describes production browser assets.
- `bootstrap/ssr/ssr.js` is the standalone Inertia renderer.

Install the plugin with:

```shell
pnpm add -D @lemmego/vite
```

The generated project already includes it. The plugin currently targets Vite 8.

## Package Frontend Contributions

Go modules can ship a `lemmego.frontend.json` manifest containing pages and
other frontend metadata. Before Vite starts, the CLI discovers manifests from
compiled dependencies and writes `.lemmego/frontend.generated.json`:

```shell
lemmego frontend sync
```

The plugin exposes the result through virtual modules:

```ts
import { pages } from 'virtual:lemmego/pages';
import { modules } from 'virtual:lemmego/modules';
```

Page keys are Inertia names such as `Scheduling/Shifts` and their values are
lazy import functions. Applications should combine them with their local page
glob and resolve them with:

```ts
import { resolvePageComponent } from '@lemmego/vite/inertia-helpers';
```

The declarations for these virtual modules are included by
`@lemmego/vite/client`. The generated `.lemmego/` directory is build metadata
and should remain ignored by Git.

## Development

Run the dev server:

```shell
npm run dev
```

This starts Vite's dev server. The `lemmego dev` command automatically detects `package.json` and starts Vite alongside air, templ and — for an Inertia project with a built SSR bundle — the Inertia renderer.

### Server-side rendering and the dev server

Vite can render your Inertia pages itself, on `/__inertia_ssr`. It is
convenient, because there is nothing to build, and slow, because it compiles
your pages on demand the first time anyone asks.

So when something else is already rendering, the generated `vite.config.js`
turns that endpoint off:

```js
inertia(command === "build" || !process.env.INERTIA_SSR_URL ? {} : { ssr: false })
```

`lemmego dev` sets `INERTIA_SSR_URL` when it starts the standalone renderer,
so this looks after itself. Leaving the endpoint on costs more than an unused
route: Vite loads every page component when the dev server starts, on the same
thread that serves your stylesheet, and your first page load waits behind it.

The `command === "build"` half matters too — the plugin is what builds
`bootstrap/ssr/ssr.js` in the first place, so it stays fully enabled for
`vite build --ssr`.

### Hot Module Replacement

Changes to `resources/js/` and `resources/css/` files are instantly reflected in the browser without a full page reload.

## Production Build

Build assets for production:

```shell
lemmego build
```

Or run Vite directly:

```shell
npm run build
```

The built assets are output to `public/build/` with a `manifest.json`:

```json
{
  "resources/js/app.tsx": {
    "file": "assets/app-abc123.js",
    "isEntry": true,
    "src": "resources/js/app.tsx"
  },
  "resources/css/app.css": {
    "file": "assets/app-xyz789.css",
    "src": "resources/css/app.css"
  }
}
```

## Root Layout

The Inertia root layout is in `resources/views/root.html`:

```html
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title inertia>{{ .appName }}</title>
    {{ .inertiaHead }}
</head>
<body>
    {{ .inertia }}
    @vite('resources/js/app.tsx')
</body>
</html>
```

## Tailwind CSS

The generated project includes Tailwind CSS v4 configured via the Vite plugin. Styles are in `resources/css/app.css`:

```css
@import "tailwindcss" source("../");

.input { @apply border rounded px-3 py-2; }
.btn-primary { @apply bg-blue-500 text-white px-4 py-2 rounded; }
```

### Where Tailwind looks for class names

`source("../")` is doing real work, so leave it there. Without it Tailwind
scans your whole project for class names, and a Lemmego project fills up with
files it has no reason to read — the file cache driver writes one per entry
under `storage/`, and a real application reached 83,000 of them. Tailwind read
all of them on every stylesheet build, which in development means every time
Vite invalidates the module. Measured at a 375px viewport: **9.6 seconds per
build, against 0.054 seconds with the scan narrowed.**

If your stylesheet suddenly takes seconds to load in development, this is the
first thing to check.

Keep classes somewhere outside `resources/`? Add it:

```css
@import "tailwindcss" source("../");
@source "../../internal/views";
```

Reaching for `@source not` to exclude a directory won't help — it filters what
the scan returns rather than stopping the scan, and measures slower than doing
nothing.

## Package Manager

The project uses `pnpm` by default with a `pnpm-workspace.yaml` for monorepo support. You can use `npm` or `yarn` as alternatives.

## Next steps

- [Deployment](/docs/deployment/) — building assets for production.
- [CLI & generators](/docs/cli/) — the `ssr` and build commands that drive it.
