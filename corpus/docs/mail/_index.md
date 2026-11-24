---
title: Mail
type: docs
prev: docs/queue
next: docs/notifications/
weight: 127
sidebar:
  open: true
---

`lemmego/mail` sends transactional email — a verification link, a password
reset, a weekly digest. A *mailable* describes the message, a renderer builds
the HTML and text bodies, a transport sends it, and
[`tasker`](/docs/queue) keeps it off the request.

## Install

```shell
go get github.com/lemmego/mail
```

```go
import "github.com/lemmego/mail"

type Welcome struct{ User *models.User }

func (w Welcome) Build(ctx mail.Context, r mail.Renderer) (*mail.Message, error) {
    html, text, err := r.Render("welcome", w)
    if err != nil {
        return nil, err
    }
    return &mail.Message{
        To:      []mail.Address{{Name: w.User.Name, Email: w.User.Email}},
        Subject: "Welcome to Example",
        HTML:    html,
        Text:    text,
    }, nil
}

// in a handler
return mail.Queue(c.RequestContext(), Welcome{User: user})
```

## Register the provider

```go
&mail.Provider{
    From: mail.Address{Name: "Example", Email: "hello@example.com"},
}
```

{{< callout type="info" >}}
**With no `Transport` set, mail is written to the log rather than sent.** This
keeps a fresh checkout from emailing real people the first time a fixture is
seeded. It warns in production, so an unset transport cannot quietly become
the reason nothing arrives.
{{< /callout >}}

To actually send, pass a transport:

```go
&mail.Provider{
    From: mail.Address{Name: "Example", Email: "hello@example.com"},
    Transport: smtp.New(smtp.Config{
        Host:     config.MustEnv("MAIL_HOST", ""),
        Port:     config.MustEnv("MAIL_PORT", 587),
        Username: config.MustEnv("MAIL_USERNAME", ""),
        Password: config.MustEnv("MAIL_PASSWORD", ""),
    }),
}
```

The provider does not build an SMTP transport from configuration on your
behalf. Pass one explicitly: a transport that defaults itself into existence
means a misconfigured `MAIL_HOST` is discovered by whoever receives the mail.

## Sending and queueing

```go
mail.Send(ctx, mailable)    // delivers on the calling goroutine
mail.Queue(ctx, mailable)   // hands it to tasker
```

**Prefer `Queue` for anything triggered by a request.** An SMTP conversation
is several round trips to someone else's server, and a relay having a slow
minute should not become a slow minute for the person who just signed up.

`Queue` needs `lemmego/queue` registered and a worker running. The provider
registers the job on **every** process, worker included — a worker needs the
factory to rebuild the job, and the same provider list boots both.

```go
mail.Queue(ctx, digest, tasker.WithDelay(10*time.Minute))
mail.Queue(ctx, digest, tasker.OnQueue("mail"))
```

## Three interfaces

```go
type Mailable interface {
    Build(ctx Context, r Renderer) (*Message, error)
}

type Renderer interface {
    Render(name string, data any) (html, text string, err error)
}

type Transport interface {
    Send(ctx Context, m *Message) error
}
```

A **mailable** describes the message once, in one place, so the subject and
the template are not scattered across handlers.

The renderer is **passed into `Build`** rather than resolved from a container.
That is what makes a mailable a plain value you can construct and assert on in
a test without booting an application.

## Templates

```
resources/mail/
  welcome.html
```

```go
mail.NewRenderer("resources/mail")        // from disk; the default
mail.FSRenderer(embedded, "mail")         // from an embed.FS
```

Override the directory with `Provider.Templates` or `MAIL_TEMPLATES`.

**A plain-text alternative is generated from the HTML** when the mailable does
not supply one, via `mail.TextFromHTML`. It keeps link targets — a text part
reading "click here" with no URL is worse than no text part — and collapses
whitespace, which HTML has plenty of and a text email should not.

## Validation

A message is checked before a transport is dialled, so a bug surfaces as an
error in your logs rather than as a connection to a relay that rejects it —
or accepts it.

| Refused | Why |
|---|---|
| No recipient | Nothing to send to |
| No subject | A blank subject line is always a bug |
| **Neither body** | An empty message is always a bug, and a relay that accepts it has published the bug to a real person |
| Invalid recipient address | Caught here rather than mid-conversation |
| No sender anywhere | Not the mailable's, not the mailer's default |

Non-ASCII headers are RFC 2047 encoded, so a subject with an em dash or a name
with an accent arrives readable rather than as mojibake.

## SMTP configuration

```go
smtp.Config{
    Host:       "smtp.example.com",
    Port:       587,          // default
    Username:   "…",
    Password:   "…",
    Encryption: "starttls",   // "tls" (465, 2465), "starttls" (587, 2587), "none"
    Timeout:    30 * time.Second,
}
```

`Encryption` is inferred from the port when unset: `tls` for 465 and 2465,
`starttls` otherwise.

{{< callout type="warning" >}}
**The alternate ports matter more than they look.** A cloud platform that
blocks outbound SMTP to stop its customers sending spam blocks 25, 465 and
587 — so 2465 and 2587 are what every relay publishes for people deploying
there, and exactly what somebody reaches for after a dial to 465 times out.

Guessing STARTTLS for 2465 is the bad failure: the server is waiting for a TLS
ClientHello and gets a plaintext EHLO, so there is no clean refusal to read.
It hangs until something times out, which looks identical to the blocked port
that sent you there.
{{< /callout >}}

`Timeout` bounds the whole conversation, because a relay having a slow minute
must not hold a worker indefinitely.

`InsecureSkipVerify` exists for a local relay with a self-signed certificate
and for nothing else. Against a real relay it makes the TLS pointless, since
an interceptor can present any certificate and read every message.

## Testing

```go
import "github.com/lemmego/mail/mailtest"

recorder := mailtest.NewRecorder()
a := apptest.New(t, apptest.WithProviders(&mail.Provider{Transport: recorder}))

// …trigger the thing that sends…

if recorder.Count() != 1 {
    t.Fatalf("expected one message, got %d", recorder.Count())
}
if got := recorder.To("grace@example.com"); len(got) != 1 {
    t.Fatal("nothing went to grace")
}
```

`Recorder` is safe for concurrent sends, which matters the moment a test
exercises a worker. `Messages`, `Count`, `Last`, `To` and `Reset` are the whole
surface.

## The log transport

`mail.NewLogTransport()` is what you get by default.

Outside production it logs the **rendered bodies** too, because there the log
is the only inbox there is — a verification email whose link is never printed
looks like a broken feature rather than an unconfigured transport. In
production it records the envelope only: those links are working credentials,
and a log is read by more people and kept far longer than an inbox.

`MAIL_LOG_BODIES` overrides the decision in either direction.

```shell
MAIL_LOG_BODIES=false   # a staging box that must not log credentials
```

{{< callout type="warning" >}}
Mail is queued, so it is the **worker** that writes these records, not the web
process. If nothing appears after a sign-up, check that a worker is running
before suspecting the mailer.
{{< /callout >}}

## Writing in the reader's language

Put a translated body in a subdirectory named after the language, beside the
one you already have:

```
resources/mail/
  verify_email.gohtml
  fr/verify_email.gohtml
```

and tell the provider where to find the language:

```go
&mail.Provider{Locale: i18n.LocaleOf}
```

A template you have not translated yet falls back on its own, so you can
translate one body at a time. A hand-written `.gotxt` text part in the base
directory is still used.

See [Localization](/docs/localization#translating-email) for the whole picture,
including mailing somebody in the language *they* chose.

## Limitations

One transport at a time. No multi-transport routing, no attachments, no bounce
handling. For multi-channel delivery — an email *and* an in-app record from one
call — see [Notifications](/docs/notifications/).

## Next steps

- [Queue](/docs/queue) — the worker that `mail.Queue` hands messages to.
- [Notifications](/docs/notifications/) — sending over several channels at once.
