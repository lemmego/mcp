---
title: Core Concepts
type: docs
prev: docs/configuration
next: docs/core-concepts/application-bootstrap
weight: 20
sidebar:
  open: true
---

Everything your application does starts with one object. It boots, collects
the things your code will need — a database connection, a cache, a session
store — and hands them to your handlers once requests start arriving. This
section is that wiring: how an application starts, how services find each
other, and how a module adds itself to your project.

If you would rather see a request served first, read [the HTTP
layer](/docs/http/) and come back. Do come back before you add your first
module, though: every module arrives as a provider, and the pages here are
what a provider is.

{{< cards >}}
{{< card link="application-bootstrap" title="Application Bootstrap" subtitle="The app lifecycle and bootstrapping pipeline" >}}
{{< card link="service-container" title="Service Container" subtitle="Service registration and resolution" >}}
{{< card link="provider-plugin-system" title="Provider & Plugin System" subtitle="Extending the framework through providers" >}}
{{< card link="dependency-injection" title="Dependency Injection" subtitle="Type-safe DI with three lifetimes" >}}
{{< card link="event-system" title="Event System" subtitle="Lifecycle events and pub/sub communication" >}}
{{< /cards >}}
