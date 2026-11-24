## When to use this skill

When sending a notification over several channels — an in-app row, an email, a
live update — or when one channel of a notification silently did not arrive.

## Send after the transaction commits, never inside it

This is the one rule the library cannot enforce for you. Inside a transaction
the broadcast fires before the row is visible, so a client that refetches
finds nothing — and if the transaction then rolls back, somebody has been told
about a notification that does not exist.

And fire only on a **real state transition**. An upsert written
`ON CONFLICT DO NOTHING` discards whether anything changed, so a notification
at that call site re-fires on every double-tap. Return whether a row was
actually inserted and send on that.

## The two interfaces

A recipient is one method:

    func (u *User) NotifiableKey() (kind, id string) { return "user", u.GetID() }

That pair is the whole contract. Everything else a channel needs — an email
address, a broadcast channel name — is a **separate optional interface
declared beside that channel**, so the core never learns what an email address
is.

A notification names its channels:

    func (n *Followed) Via(to notify.Notifiable) []string {
        return []string{database.Name, broadcast.Name}
    }

Two things about `Via`. Returning `nil` is a legitimate answer, not an error —
somebody who muted this kind of notification gets no channels. And **the order
is honoured**: name `database` before `broadcast` so the row exists before
anything tells a browser to go and look for it.

## Use the channel packages' constants

`database.Name`, not `"database"`. Go has no duck typing, so a `Via` that
names a channel its notification cannot satisfy is a **runtime** failure. The
constants make a typo a build error, and the core defines no channel names at
all.

A channel that cannot satisfy a notification returns a named error rather than
skipping silently:

    *Followed says via "mail" but does not implement mail.Mailable (missing ToMail)

## Write the contract test

The moment a project has a second notification, it owes itself this:

    func TestNotificationsSatisfyTheirChannels(t *testing.T) {
        notifytest.Contract(t, notifier, notifytest.User{ID: "1"},
            notifications.All()...)
    }

`All()` is three lines in your own notifications package returning a zero value
of every type. **A notification missing from it is not checked**, which is why
the list and the types live in the same package where a reviewer sees both.

## Wiring

Channels are constructed by the application and handed over:

    conn, _ := db.Resolve(a)
    store, err := sqlstore.New(conn)

    &provider.Provider{
        Channels: []notify.Channel{
            database.New(store),
            mail.New(lmail.MustGet(a)),
            broadcast.New(hubAdapter{hub}),
        },
    }

That is what keeps this module from importing a SQL driver, an SMTP client or a
WebSocket library to offer them. The provider goes after the database connector
and after mail.

## The migration is yours

    lemmego publish --tags=notify-migrations

It writes into your own migrations directory, and is then yours to read, edit
and roll back.

## Limits worth knowing

- The SQL store ships statements for **Postgres and SQLite only** and refuses
  anything else rather than emitting untested DDL. There is no MySQL store.
- `ErrNoNotifier` means nothing is installed; `ErrNoChannel` means `Via` named
  something that is not registered.

## Check first

`search-docs` section `notifications`, `module-api` on `notify`.
