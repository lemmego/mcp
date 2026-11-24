## When to use this skill

When an application talks to a database through the Go Persistence API rather
than against one ORM directly — or when you are deciding whether it should.

## Decide this first: GPA or the ORM directly

GPA buys one thing — the same repository code over GORM, Bun, MongoDB, Redis or
the Lemmego ORM. It costs expressiveness: anything the adapter cannot express
returns `gpa.ErrorTypeUnsupported`.

So use `lemmego/orm` directly unless something concrete needs the abstraction.
A project that will only ever run on one SQL database pays the cost and
collects nothing. See the `lemmego-orm` skill.

## Register the provider, not the handle

    provider, err := gpagorm.NewProvider(gpa.Config{ /* … */ })
    if err != nil {
        return err
    }
    gpa.Register[*gpagorm.Provider]("primary", provider)

    primary := gpa.MustGet[*gpagorm.Provider]("primary")

`Register` takes a **provider**, built by that adapter's constructor — not a
`*gorm.DB` or a `*sql.DB`. Passing the raw handle does not compile, and it is
the most common mistake here because older prose shows it.

`Get` returns an error; `MustGet` panics, which is appropriate at boot and
wrong inside a handler. `RegisterDefault` skips the instance name.

## Repositories are generic and type safe

    userRepo := gpagorm.GetRepository[User](primary)
    user, err := userRepo.FindByID(ctx, 123)   // *User, not interface{}

`GetRepositoryByName[T]("primary")` resolves the provider for you. There is no
casting and no reflection on the operation path.

## Check unsupported operations by type, not by comparison

    if gpa.IsErrorType(err, gpa.ErrorTypeUnsupported) {
        // fall back to raw SQL for this one query
    }

`gpa.ErrorType` is a string type with a set of constants — validation, not
found, unsupported, and so on. Use `IsErrorType` rather than comparing an error
value, so a wrapped error still matches.

**An unsupported operation is an honest refusal, not a silent drop.** Row
locking, subqueries and DDL generation come back as `ErrorTypeUnsupported`
rather than being quietly omitted — so handle it rather than assuming the query
ran.

One difference worth knowing between adapters: `gpaorm` **does** translate
composite AND/OR conditions, and `gpagorm` drops them. If a compound predicate
is silently not applied, that is the adapter, not your code.

## The adapters

| Provider | Backs |
| --- | --- |
| `gpaorm` | the Lemmego ORM — registers under the name `LemmegoORM` |
| `gpagorm` | GORM |
| `gpabun` | Bun |
| `gpamongo` | MongoDB |
| `gparedis` | Redis |

`gpaorm` takes an already-open `*orm.DB`, so it needs no SQL driver dependency
of its own.

## Connectors are a different thing

`ormconnector`, `gormconnector` and `bunconnector` open the connection and
publish `db.Connection` into the application. They are what the queue, the
database session driver and `observe` borrow. GPA is the query abstraction
above that; the connector is the lifecycle below it, and a project needs the
connector whether or not it uses GPA.

**Provider order:** the connector first, then anything that borrows the pool.

## Entity hooks run in a fixed order

`Validate` → `BeforeCreate`/`BeforeUpdate` → the database operation →
`AfterCreate`/`AfterUpdate`. A new adapter that gets this order wrong passes
most tests and corrupts the one case that depends on validation running first.

## Check first

`search-docs` section `database`, `module-api` on `gpa` or on the adapter.
When changing `gpa`'s interfaces, every adapter has to be updated with it.
