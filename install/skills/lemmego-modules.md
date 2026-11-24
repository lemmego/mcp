## When to use this skill

When a **tenant** has some features and not others — scheduling, patrols,
payroll — and the application has to know which, per request.

## "Module" means three things; this is the third

They nest: a **Go module** is a unit of compilation and release; a **Lemmego
module** is a published package like `mail` or `seo`; and a **module here** is
a feature a tenant has or has not.

This package is the registry of the third kind. It is not a feature flag: a
flag is on for ten percent of traffic, whereas this is the unit of a tick in a
settings screen, a line on an invoice, and a Go module an on-premise build can
leave out of the binary entirely.

If you want the second sense — what the ecosystem publishes — ask
`list-modules` instead.

## The shape

    registry, err := host.RegistryOf(plugins...)      // boot: validate the graph
    set := modules.Union(registry, enabledNames)      // per request
    admin.Get("/shifts", host.RequireModule("scheduling"), listShifts)

`RegistryOf` validates the dependency graph **at boot**, so a module enabled
without something it depends on is a startup error rather than a 500 on the one
page that needs it.

`RequireModule` is the route gate. Use it rather than checking the set inside a
handler — a gate that lives on the route is visible in the routing table, and
`list-routes` will show it.

## Three packages, split by dependency

| Package | May import | Holds |
| --- | --- | --- |
| `modules` | **standard library only** | `Module`, `Registry`, `Set`, `Term`, the graph, navigation filtering |
| `modules/store` | `orm` | per-tenant enablement, settings, term overrides, the DDL |
| `modules/host` | `api/app` | `Plugin`, `Contributors`, `RequireModule`, `Guards` |

The split is enforced by `deps_test.go` in **both** directions. Do not add an
import that crosses it: one 1.1 KB file declaring the plugin interface imported
`api/app` for it and thereby dragged a web server into the registry and into
everything importing the registry. An import added in the wrong file is
invisible until somebody profiles a binary.

## Contributions are optional interfaces

A feature module implements only the interfaces it has something to say for —
navigation, settings, terminology — rather than filling in a struct with
zero values for the rest. Check what a plugin actually implements with
`module-api` rather than assuming a full surface.

## Terminology

`Term` lets a tenant rename a concept — "shifts" to "rosters" — without a fork.
Read labels through it in any surface a tenant sees, or the rename will be
half-applied and that is more confusing than no rename.

## Check first

There is **no documentation page for this module on the site yet** — its README
is the authoritative spec. `module-api` on `modules`, `modules/store` or
`modules/host`.

It pairs with `lemmego/tenancy`, which answers *which* tenant a request is
about; this answers what that tenant has. See the `lemmego-tenancy` skill.
