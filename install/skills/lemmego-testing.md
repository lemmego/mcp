## When to use this skill

When writing or debugging tests in a Lemmego project.

## The failure this project keeps hitting

**A suite that skips is indistinguishable from one that passes in the exit
code.** `apptest.WithScratchDB()` fails rather than skipping for exactly that
reason, so an unreachable database is now loud — but the lesson generalises to
every `t.Skip` in a suite.

The giveaway is the runtime. If a package that boots the application finishes in
a second, it did nothing. Check with `go test -v` and look for `--- SKIP`.

The server comes from `.env.test`, not from an environment variable. Two ways
to name a database is two things that can disagree, and they did: the old
`APPTEST_POSTGRES_DSN` supplied host, user, password and port but not options,
so a managed URL's `sslmode=require` survived into a local server and every
test panicked.

## Verify the probe, not just the test

A new test is not trusted until it has been seen to fail:

1. Revert the code the test protects.
2. **Confirm the revert actually applied** — grep for it. A no-op patch reads
   exactly like a vacuous test.
3. Watch the test fail.
4. Restore, and re-run.

Real examples from this project: a test passed by asserting two identical
redirects were identical; another exercised a helper that predated the feature;
a third went through a code path that forced the value it was asserting on.

## Asserting on rendered output

Read the Inertia props rather than grepping the HTML. A page echoes its own
query back, so a substring check against the body is true for any query whose
text appears in a result. React also inserts `<!-- -->` between static text and an
interpolated value, so `Continue with GitHub` is emitted as
`Continue with <!-- -->GitHub` and a naive match fails.

## What the harness gives you

A real boot with every provider, a request builder, Inertia assertions,
`ActingAs`, a Postgres schema per test, and `Server()`/`Client()` when a test needs a
real connection — `httptest.NewRecorder` cannot be hijacked, so a WebSocket
upgrade fails and an SSE handler blocks forever without them.
