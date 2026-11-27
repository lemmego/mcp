---
title: Mobile Module Contributions
type: docs
prev: docs/frontend/vite
next: docs/frontend/_index
weight: 45
sidebar:
  open: true
---

An optional Go module can own the React Native screens and native configuration
for its feature. The app shell still owns authentication, workspace bootstrap,
shared persistence, and the common navigation frame.

## Add `lemmego.mobile.json` to the Go module

The Lemmego CLI reads this manifest from packages in the selected Go app build
when you run:

```sh
lemmego mobile sync
```

For example:

```json
{
  "schema": 1,
  "id": "guard_operations",
  "entry": "mobile/index.tsx",
  "requires": ["attendance"],
  "permissions": ["incidents.view", "incidents.create"],
  "routes": [
    {
      "id": "guard_operations.incidents",
      "path": "guard-operations/incidents",
      "title": "Incidents",
      "permission": "incidents.view"
    }
  ],
  "native": {
    "expoPlugins": ["mobile/withCameraPermission.js"],
    "iosInfoPlist": {
      "NSCameraUsageDescription": "Capture incident evidence."
    },
    "androidPermissions": ["CAMERA"]
  }
}
```

`entry` and every Expo config plugin path are relative to the Go module and
must name files inside it. Route IDs must be unique across the selected build.
Every `requires` module must also contribute to the selected Go build; a
distribution that omits a required module fails sync. The CLI writes a
deterministic `.lemmego/mobile.generated.json` from discovered contributions.
In that output, `root` is relative to the Go application root, while `entry`
and Expo plugin paths stay relative to that module root. Generated metadata is
disposable build output, so do not edit or commit it.

An app build consumes only the contributions listed in its generated manifest.
This keeps SaaS builds broad while letting each on-premise app build match the
modules compiled into its server. Workspace enablement and user permissions
remain runtime checks supplied by bootstrap and enforced by server routes.
