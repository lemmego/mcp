## When to use this skill

When adding or changing an Inertia page, or debugging a blank one.

## A 200 proves nothing

Inertia falls back to client rendering when the server-side render fails. The
page answers **200 and looks perfect to a human** while a crawler receives an
empty `<div id="app">"`. Check the body for `data-server-rendered="true"` and for
real markup inside the container — not the status code.

Outside production the SSR guard turns that into a 500 with the reason, which is
why development is strict and production is not.

## Never touch a browser global during render

The renderer is Node: `window`, `document` and `localStorage` all throw. A page that
read `window.location.search` while rendering took down every request to it. Pass
what you need as a prop, or read it in `useEffect`, which never runs in the
renderer.

## The title is rendered in Go

`lemmego/seo` writes the real `<title>` server-side, so Inertia's `<Head title>`
emits a **second** one. Update the tab title in an effect instead.

## Props and auth ordering

Shared props are resolved before the route's own middleware, so an eagerly
computed prop sees a signed-out viewer even for a signed-in person. Pass a
`func() any` closure so it resolves after auth.
