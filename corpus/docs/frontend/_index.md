---
title: Frontend
type: docs
prev: docs/http/context
next: docs/frontend/go-templates
weight: 40
sidebar:
  open: true
---

Your handler has some data and now has to turn it into a page somebody can
look at. This section is about that step.

There are two broadly different ways to do it, and the choice shapes how the
rest of your frontend feels.

**Build the HTML on the server.** Your handler fills a template with data and
sends finished HTML. The browser receives a complete page; clicking a link
fetches another complete page. This is how the web worked first, it is simple
to reason about, and it needs no JavaScript at all.

**Build it in the browser.** Your server sends data, and JavaScript running in
the page turns it into HTML and swaps pieces out as somebody clicks around —
so the page never goes blank between views. That is a **single-page
application**, or SPA. You get a smoother feel and pay for it with a
JavaScript build step and a second language in your project.

Lemmego supports both, plus a middle road that is usually the best of the
three — Inertia, which gives you the SPA feel while your routing and data
loading stay in Go.

### Your options

| Approach | What you write | Choose it when |
|---|---|---|
| **Go templates** | HTML files with `{{ }}` holes | You want the simplest thing that works, and no Node |
| **Templ** | Templates that compile to Go | You want a typo in a field name to be a build error, not a blank space |
| **Inertia** | React or Vue components | You want an app-like feel without building a separate API |

{{< callout type="info" >}}
You chose one of these when you ran `lemmego new`, and it is already wired up.
Read the page for yours; you can ignore the other two.
{{< /callout >}}

### Asset pipeline

All frontend approaches work with the Vite asset pipeline for bundling JavaScript, CSS, and hot module replacement during development.

{{< cards >}}
{{< card link="go-templates" title="Go Templates" subtitle="Server-rendered HTML with layouts and partials" >}}
{{< card link="templ" title="Templ" subtitle="Type-safe Go templates" >}}
{{< card link="inertia" title="Inertia.js" subtitle="Modern SPA with React or Vue" >}}
{{< card link="vite" title="Vite Integration" subtitle="Asset pipeline and HMR" >}}
{{< /cards >}}
