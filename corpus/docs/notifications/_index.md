---
title: Notifications
type: docs
prev: docs/mail/
next: docs/broadcast/
weight: 128
sidebar:
  open: true
---

`lemmego/notify` delivers one notification over several channels at once —
storing a row your application can list, sending an email, and pushing to an
open browser connection, from a single `Send` call.

A notification is a type that names the channels it goes out on. Each channel
then asks, by type assertion, whether the notification implements the one
interface that channel needs.

## Install

```shell
go get github.com/lemmego/notify
```

## Define a notification

```go
type Followed struct {
    ActorID     uint64
    ActorHandle string
}

func (Followed) Via(notify.Notifiable) []string {
    return []string{database.Name, broadcast.Name}
}

// database
func (Followed) NotificationType() string { return "followed" }
func (f Followed) ToDatabase(context.Context, notify.Notifiable) (any, error) {
    return map[string]any{"actor_id": f.ActorID, "handle": f.ActorHandle}, nil
}

// broadcast
func (Followed) BroadcastEvent() string { return "notification.created" }
func (f Followed) ToBroadcast(context.Context, notify.Notifiable) (any, error) {
    return map[string]any{"type": "followed"}, nil
}

// Assert the pairing at compile time — one line per channel.
var _ database.Storable       = Followed{}
var _ broadcast.Broadcastable = Followed{}
```

`Via` receives the recipient, so a notification can return different channels
for different people — or none at all, which is how notification preferences
live in your application rather than here.

**The order of `Via` is the order of delivery.** List `database` first: it is
the record, while `mail` and `broadcast` only announce that the record exists.

## Send it

```go
notify.Send(ctx, user, Followed{ActorID: actor.ID, ActorHandle: actor.Handle})
```

{{< callout type="warning" >}}
**Send after the transaction commits, never inside it.**

Inside a transaction, the broadcast fires before the row is visible, so a
client that refetches finds nothing. If the transaction then rolls back,
somebody has been told about a notification that does not exist.

No ordering guarantee inside `Send` can protect a caller from this, so it is
the one rule the library cannot enforce for you.
{{< /callout >}}

**Send only on a real state transition.** An upsert written
`ON CONFLICT DO NOTHING` discards whether anything changed, so a notification
at that call site fires again on every repeat. Have the write report whether a
row was actually inserted, and send on that:

```go
created, err := repo.Follow(ctx, followerID, followeeID)
if err != nil {
    return err
}
if created {
    notify.Send(ctx, followee, Followed{...})
}
```

## Channels

| Channel | Package | Needs |
|---|---|---|
| `database` | `notify/channel/database` | a `Store`; standard library only |
| `database` (SQL) | `notify/channel/database/sqlstore` | an `api/db.Connection` |
| `mail` | `notify/channel/mail` | a `*mail.Mailer` |
| `broadcast` | `notify/channel/broadcast` | a four-primitive `Publisher` |

Channel names are constants in the channel packages, so `database.Name` is
checked by the compiler and a typo is a build error. The core package defines
no channel names.

## Wire it up

```go
conn, _ := db.Resolve(a)
store, err := sqlstore.New(conn)

&provider.Provider{
    Channels: []notify.Channel{
        database.New(store),
        mail.New(lmail.MustGet(a)),
        broadcast.New(hubAdapter{hub}),
    },
}
```

Your application constructs the channels and hands them over, the same way
`mail.Provider` takes its `Transport`. This is what keeps the module from
importing a SQL driver, an SMTP client or a WebSocket library in order to
offer them — an application that wants only in-app notifications compiles no
SMTP client.

Then publish the migration:

```shell
lemmego publish --tags=notify-migrations
```

The schema arrives as a migration in your project rather than being created at
boot, so it is yours to read, edit and roll back.

## Test that every notification can be delivered

Go has no duck typing, so a `Via` naming a channel its notification does not
satisfy fails at **runtime**, when a real person was supposed to be told
something. Two things catch it earlier.

A channel returns a named error rather than skipping silently:

```text
*Followed says via "mail" but does not implement mail.Mailable (missing ToMail)
```

And `notifytest.Contract` moves the check into your test suite. Add it when
you define your second notification:

```go
func TestNotificationsSatisfyTheirChannels(t *testing.T) {
    notifytest.Contract(t, notifier, notifytest.User{ID: "1"},
        notifications.All()...)
}
```

`All()` is a short function in your own notifications package returning a zero
value of every type. A notification missing from that list is not checked, so
keep the list beside the types where a reviewer sees both.

## The unread count

Read it with a capped query rather than keeping a counter column:

```sql
SELECT count(*) FROM (
    SELECT 1 FROM notifications
     WHERE notifiable_type = $1 AND notifiable_id = $2 AND read_at IS NULL
     LIMIT 101
) t
```

The cap is what makes this cheap forever: it reads at most `MaxUnread+1`
entries from a partial index whose entire size is the number of *unread*
notifications in the system, and a badge shows "99+" past that anyway.

**Do not cache it.** An unread badge is the thing the person just clicked, so
a stale one is the most noticeable bug a notification system can have.

A counter column is the tempting alternative and it goes wrong: `col = col + 1`
is not expressible through the ORM, so it needs raw SQL at every write, and it
drifts the first time anyone prunes a row or deletes one by hand. The symptom
is a badge stuck at 3 that no log explains.

## Store behaviour

The store's methods are **transitions, not CRUD**. Marking a notification read
is one conditional update whose affected row count is the answer, so two taps
on one bell cannot both report a change. The owner is part of every condition,
so one person cannot touch another's notifications even if a handler forgets to
check.

`jsonb` does not preserve the key order of your payload, because it stores a
parsed form — the same property that makes it indexable. Every key and value
survives, numbers included. A payload with duplicate keys is collapsed, so do
not build one.

## Mailing somebody in their own language

A recipient says which language it wants by implementing one method:

```go
func (u *User) PreferredLocale() (string, bool) {
    return u.Locale, u.Locale != ""
}
```

Return `false` for "this person has never chosen" — that is a state, not a
failure, and they get the default language.

Then tell the mail channel how to carry it:

```go
mailchan.New(mailer,
    mailchan.WithLocale(func(ctx context.Context, locale string) context.Context {
        return i18n.With(ctx, localizer.For(locale))
    }),
)
```

If you implement `PreferredLocale` and forget `WithLocale`, the notification
still sends in the default language and you get one warning naming the type —
rather than silence, which is the worst outcome for a message delivered in the
wrong language.

See [Localization](/docs/localization#translating-email).

## Limitations

- **Notification preferences** — `Via` takes the recipient, so they belong in
  your application alongside the product decisions.
- **Digests and grouping** — the `dedupe_key` column is sized to serve as a
  collapse key when you need one.
- **`notify.Queue`** — `Send` is synchronous. The only slow channel is mail,
  and [`mail.Queue`](/docs/mail/) already renders on the calling goroutine and
  enqueues a self-contained message, so deferral happens where it is needed.
- **Routes and an inbox UI** — `database.Inbox` gives you the five methods;
  the page is yours.
- **MySQL DDL** — refused by name rather than emitted untested.
- **Channel-level retry** — [`tasker`'s](/docs/queue) `ShouldRetry` owns that.

## Next steps

- [Broadcasting](/docs/broadcast/) — the live connection the `broadcast`
  channel publishes to.
- [Mail](/docs/mail/) — configuring the transport the `mail` channel uses.
