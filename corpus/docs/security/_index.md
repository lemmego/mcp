---
title: Security
type: docs
prev: docs/observe/
next: docs/security/authentication
weight: 100
sidebar:
  open: true
---

How a request proves who it is, and what it may then do.

Start with **Authentication** to set up sign-in, then **The current user** for
reading the signed-in user in a handler. **CSRF protection** is already on in
a scaffolded project; read it when a form starts returning 419.

{{< cards >}}
{{< card link="authentication" title="Authentication" subtitle="Session & JWT auth with middleware" >}}
{{< card link="current-user" title="The Current User" subtitle="One type on every authenticated path" >}}
{{< card link="csrf" title="CSRF Protection" subtitle="Cross-site request forgery prevention" >}}
{{< card link="encryption" title="Encryption" subtitle="AES-256-GCM encryption" >}}
{{< card link="/docs/oauth2" title="OAuth2 Server" subtitle="Issue and revoke tokens for third-party apps" >}}
{{< card link="/docs/rbac" title="Roles and Permissions" subtitle="Database-backed roles, checked through authz" >}}
{{< card link="/docs/tenancy" title="Multi-tenancy" subtitle="Which tenant a request is about, and keeping rows apart" >}}
{{< /cards >}}
