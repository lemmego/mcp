## When to use this skill

When sending email: a welcome message, a password reset, a digest.

## `Send` on the request, `Queue` off it

    mail.Send(ctx, mailable)    // delivers on the calling goroutine
    mail.Queue(ctx, mailable)   // hands it to tasker

**Prefer `Queue` for anything triggered by a request.** An SMTP conversation is
several round trips to someone else's server, and a relay having a slow minute
should not become a slow minute for the person who just signed up.

`Queue` needs `lemmego/queue` registered and a worker running
(`lemmego run tasker:work`). The provider registers the job on **every**
process, worker included — a worker needs the factory to rebuild the job, and
the same provider list boots both. So do not guard the mail provider behind
"only in the web process".

`Queue` takes `tasker.DispatchOpt`:

    mail.Queue(ctx, digest, tasker.WithDelay(10*time.Minute))
    mail.Queue(ctx, digest, tasker.OnQueue("mail"))

## Three interfaces, and one of them is the design

    type Mailable interface {
        Build(ctx Context, r Renderer) (*Message, error)
    }
    type Renderer interface {
        Render(name string, data any) (html, text string, err error)
    }
    type Transport interface {
        Send(ctx Context, m *Message) error
    }

A mailable describes the message once so the subject and the template are not
scattered across handlers.

**The renderer is passed into `Build`** rather than resolved from a container.
That is what makes a mailable a plain value you can construct and assert on in
a test without booting an application — so write the test against `Build`, not
against a sent message.

## A message is validated before a transport is dialled

`ErrNoRecipient` and its siblings come back before any network call, so a
malformed message fails fast and locally rather than after a connection
timeout. Do not add your own pre-flight checks for the same things.

## Text bodies

`TextFromHTML(html)` derives a plain-text part. Send both: a text-only client
shows an empty message otherwise, and a missing text part is a measurable spam
signal.

## Testing

`NewLogTransport()` records instead of sending. Use it in tests and in local
development — it is better than a real relay pointed at a catch-all, because
the assertion is on a value rather than on somebody's inbox.

## Check first

`search-docs` section `mail`, `module-api` on `mail`. For multi-channel
delivery — an in-app row plus an email plus a live update — the module is
`notify` and mail becomes one of its channels; see the `lemmego-notify` skill.
