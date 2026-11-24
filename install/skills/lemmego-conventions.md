## When to use this skill

Before adding a handler, a model, a route, a repository or a service to a
Lemmego project. It is about **where code goes and how thin each layer stays**,
which no function signature can tell you.

## Generate, do not hand-write

The generators decide the filename, the package and the shape, which is most
of what this skill is about:

    lemmego gen model post            # internal/models/post.go
    lemmego gen handlers post         # internal/handlers/post_handlers.go
    lemmego gen migration create_posts
    lemmego gen input post            # request validation
    lemmego gen form post             # frontend form

`gen` is aliased to `g`, and each subcommand to its first letter, so
`lemmego g h post` is the same command. Run with no name to be prompted.

**The paths are the project's to change, not yours to assume.** They come from
the application itself — `lemmego run config:paths` prints them — and fall back
to the conventional layout below. Read them rather than hardcoding
`internal/handlers`.

## The layout

    cmd/app/              entry point and CLI commands
    bootstrap/
      providers.go        LoadProviders() — order is load-bearing
      routes.go           LoadRoutes() — a list of route callbacks, nothing more
    internal/
      handlers/           HTTP handlers, resourceful only
      models/             one model per file
      repos/              database access
      routes/             one file per concern
      middleware/         application middleware
      migrations/         schema
      commands/           CLI commands
    resources/            js, css, views
    public/               built assets

## One model per file

`lemmego gen model post` writes `internal/models/post.go` holding one struct.
Keep it that way: a second model in that file is a model nobody finds by
guessing the filename, and the generator will not edit it for you.

Models are **plain structs**. They carry tags, hooks and relationships — never
a database handle. A model holding a live connection cannot be serialised,
copied between goroutines, or built in a test.

## Handlers hold the seven resourceful methods, and nothing else

`lemmego gen handlers post` writes exactly these:

    func PostIndexHandler(ctx app.Context) error    // list
    func PostCreateHandler(ctx app.Context) error   // the "new" form
    func PostShowHandler(ctx app.Context) error     // one record
    func PostStoreHandler(ctx app.Context) error    // create it
    func PostEditHandler(ctx app.Context) error     // the "edit" form
    func PostUpdateHandler(ctx app.Context) error   // save it
    func PostDeleteHandler(ctx app.Context) error   // remove it

Note `app.Context` **by value**, not `*app.Context`. The pointer form appears
in some older prose and does not compile.

**An action that is not one of those seven is a sign you want another
resource.** `PostPublishHandler` wants to be `PublicationStoreHandler`;
`UserResetPasswordHandler` wants to be a `PasswordReset` resource with `Create`
and `Store`. Reach for a new resource before adding an eighth method — that is
what keeps a handler file something a reader can predict.

## Handler methods stay lean: parse, delegate, respond

A handler does three things. If it is doing a fourth, that fourth thing
belongs somewhere else.

    func PostStoreHandler(ctx app.Context) error {
        // NewPostInput parses the body and validates it in one call — it
        // picks a decoder from the Content-Type and returns the validation
        // error, so a handler never does either itself.
        in, err := inputs.NewPostInput(ctx)
        if err != nil {
            return err
        }

        post, err := services.PublishPost(ctx, in)
        if err != nil {
            return err
        }

        return ctx.JSON(app.M{"post": post})
    }

Generate that input with `lemmego gen input post`. It writes
`NewPostInput(c app.Context)`, which calls `c.ParseInput` and then its own
`Validate()` built on `c.Validator()`.

What must **not** be in a handler:

- **SQL, or an ORM query.** That is the repository's job.
- **A transaction spanning more than one repository call.** That is a service.
- **Business rules** that another caller — a CLI command, a queued job, a
  second handler — would need. The moment a rule lives in a handler, the job
  that needs it reimplements it slightly differently.

Return the error rather than rendering it. The framework's error handling turns
a typed error into the right status; a handler that writes its own 500 loses
that.

## Database work goes in a repository

A repository is a struct holding the handle, with methods that take a context:

    package repos

    type PostRepository struct {
        db *orm.DB
    }

    func Post(a app.App) *PostRepository {
        return &PostRepository{db: getDB(a)}
    }

    func (r *PostRepository) Published(ctx context.Context) ([]models.Post, error) {
        return r.db.Model[models.Post]().
            Where(orm.Eq("published", true)).
            OrderBy(orm.Desc("created_at")).
            All(ctx)
    }

Two things about this that are easy to get wrong:

- **There is no `lemmego gen repo`.** The layer is a convention, established by
  the generated authentication code in `internal/repos/`. Read
  `internal/repos/user_repo.go` in this project and match it; do not invent a
  different shape beside it.
- **The constructor takes `app.App` and resolves the handle**, rather than the
  handle being passed in from the handler. That is what keeps handlers from
  knowing which ORM this project uses.

## Services are for work that spans repositories

A service is where a transaction, two repositories, or a repository plus an
outbound call belong — anything that is a *use case* rather than a query.

The scaffold does not create `internal/services/` and nothing generates it, so
if this project has no such directory, creating one is a decision: say so
rather than quietly introducing a layer. A single-repository call does not need
a service, and a service that only forwards to one repository is a file that
makes every change twice as long to read.

## Route files stay a table of contents

`internal/routes/<concern>.go` holds one exported function per concern, and
`bootstrap/routes.go` lists them:

    // internal/routes/posts.go
    func PostRoutes(a app.App) {
        r := a.Router()
        posts := r.Group("/posts")
        {
            posts.Get("/", handlers.PostIndexHandler)
            posts.Get("/{id}", handlers.PostShowHandler)
            posts.Post("/", auth.Protected, handlers.PostStoreHandler)
        }
    }

    // bootstrap/routes.go
    func LoadRoutes() []app.RouteCallback {
        return []app.RouteCallback{
            routes.WebRoutes,
            routes.ApiRoutes,
            routes.PostRoutes,
        }
    }

A route file declares **paths, middleware and the handler to call**. An inline
closure with a body is acceptable for a one-line health check or a ping; it is
not acceptable for anything with a branch in it. If a route's closure has grown
an `if`, it has become a handler and belongs in `internal/handlers`.

Add a new concern as a new file plus one line in `LoadRoutes()`, rather than
growing `ApiRoutes` — a route file that needs scrolling is one nobody reads
before adding a duplicate path.

## Before you write any of it

- `list-routes` — what this application already serves, so you do not add a
  second route for the same path.
- `list-modules` — whether a published module already does what you are about
  to build. There are modules for multi-tenancy, roles and permissions,
  translation, search, billing and more, and `module-api` cannot see them until
  they are in `go.mod`.
- `module-api` — the real signatures, before calling anything you have not read
  in this session.
