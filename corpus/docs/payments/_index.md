---
title: Payments
type: docs
prev: docs/seo/
next: docs/localization/
weight: 134
sidebar:
  open: true
---

`lemmego/pay` handles subscription billing: one hosted checkout, one signed
webhook, and an optional reader for verified revenue. It ships a Stripe driver
over `net/http` with no SDK.

```go
&pay.Provider{
    Gateway: stripe.New(stripe.Config{
        SecretKey: config.MustEnv("STRIPE_SECRET_KEY", ""),
    }),
    Store:   func(a app.App) pay.Store { return billing.NewStore(a) },
    Secrets: []string{config.MustEnv("STRIPE_WEBHOOK_SECRET", "")},
}

r.Post("/webhooks/stripe", pay.WebhookHandler())
```

## Install

```shell
go get github.com/lemmego/pay
```

## Three rules to build on

The expensive mistakes in billing are not in the API calls — they are in what
your application believes afterwards. These three shape everything below.

**1. Grant entitlement from the webhook, never from the redirect.** A browser
returning to a success URL proves only that a browser was sent there: the
person can close the tab, the redirect can be lost, and the URL can be visited
again tomorrow by anyone who remembers it. The webhook is signed, retried
until acknowledged, and is the only thing `Handle` will write from.

**2. Expect every delivery more than once, and out of order.** Stripe retries
until it gets a 2xx and promises no ordering. So `Store.Seen` deduplicates on
the event id, and `Store.Apply` must let an older event lose to a newer one
**as a condition on the write, not a read followed by a write** — a
read-then-write guard has a window exactly wide enough to lose the newer
event.

**3. Verify the signature over the exact bytes received**, with the timestamp
inside the MAC and a tolerance around it. Without the tolerance, a captured
signature is valid forever; without the raw bytes the check passes only by
luck, because decoding and re-encoding the same JSON changes the signature.

{{< callout type="info" >}}
**With no `Gateway` configured, paid paths refuse with `ErrNoGateway` and the
rest of the application runs normally**, so a checkout without Stripe
credentials is not a prerequisite for rendering a home page.
{{< /callout >}}

Leave `Livemode` unset and it is derived from the gateway. A flag that can
disagree with the key in use eventually does, and that failure is a test-mode
event granting a real subscription.

## Money

```go
type Money struct {
    Amount   int64  // in the currency's smallest unit
    Currency string
}
```

Always an integer in the currency's smallest unit — never a float. A
`float64` holds `0.1` as
`0.1000000000000000055511151231257827`, and a sum of a few thousand
subscription amounts drifts by enough to show the wrong figure — which for a
badge calling itself *verified* is the whole value gone.

{{< callout type="warning" >}}
**Convert with `Factor(currency)`, never by dividing by 100.** Zero-decimal
currencies are reported in whole units: Stripe sends a ¥5,000 subscription as
`5000`, so dividing by 100 displays ¥50 — wrong by two orders of magnitude.
{{< /callout >}}

```go
m.Major()            // whole units, truncated
m.Add(other)         // refuses mixed currency: ErrMixedCurrency
m.Sub(other)         // floors at zero
m.DivRound(n)        // rounds half away from zero
m.Percent(17.5)
```

There is no exchange rate here, and adding one would undermine the point: a
rate is a time-varying external fact, so a revenue figure that depends on one
cannot be verified by the person reading it.

## Subscription status

```go
s.Status.Entitling()   // active, trialing — may they use the feature?
s.Status.Earning()     // active, past_due — does this count as revenue?
```

The two differ on `past_due`. Somebody whose card has just failed still counts
as revenue and should be prompted rather than cut off, so entitlement is the
stricter of the two questions.

## Grace period

```go
const pay.GracePeriod = 48 * time.Hour
```

A missed renewal webhook — a deploy during a delivery, a 500 from your own
handler — should cost two days of grace rather than revoking a paying
customer's access. Check entitlement through a helper that applies it:

```go
func (u *User) ProActive() bool {
    if !u.IsPro {
        return false
    }
    if u.ProUntil == nil {
        return true
    }
    return time.Now().UTC().Before(u.ProUntil.Add(pay.GracePeriod))
}
```

Keep the expiry check. Without it, a subscription that was never cancelled by
a webhook grants access forever.

## The webhook

| Outcome | Status | Why |
|---|---|---|
| Verified and applied | 200 | |
| Already seen | 200 | A retry must do nothing, and must not be retried again |
| Event this app ignores | 200 | Acknowledged, not an error |
| Bad signature | **400** | Not 500 — nothing to retry |
| Store failed | **500** | So Stripe retries. Never acknowledge a write that did not happen |
| Test-mode event in live mode | refused | A free subscription otherwise |

Exempt the route from CSRF, and write the reason next to the exemption — the
request is signed, which is a stronger check than a CSRF token, and Stripe
cannot send one.

## Implementing `Store`

```go
type Store interface {
    Seen(ctx Context, event *Event) (first bool, err error)
    Apply(ctx Context, event *Event) error
}
```

`Seen` should be a unique index on the event id and nothing more.

`Apply` is one statement:

```sql
INSERT INTO subscriptions (…, event_at) VALUES (…)
ON CONFLICT (id) DO UPDATE SET …, event_at = excluded.event_at
WHERE excluded.event_at >= subscriptions.event_at
```

The `WHERE` on the `DO UPDATE` is what makes rule 2 hold: the guard is applied
by the database, so there is no window between the check and the write.

## Verified revenue

`RevenueReader` is a separate interface from `Gateway`, so an application can
charge customers without also being able to read anyone's books. The two are
different authorisations with different blast radii.

```go
revenue := paystripe.NewRevenue(gateway, clientID, clientSecret)
url, err := revenue.AuthorizeURL(state, redirectURI)   // read_only; state required
account, err := revenue.Exchange(ctx, code)            // stores only the account id
total, err := pay.Total(ctx, revenue, account, time.Now())
```

Connect is asked for `read_only` and **the returned scope is checked**, not
assumed. No access token is stored — only the account id.

### Publishing a band rather than a figure

```go
band, ok := pay.BandFor(total.Monthly)
```

The ladder is coarse at the bottom and open at the top. The useful distinction
for a reader is the order of magnitude, and a ladder fine enough to separate
$4,900 from $5,100 reveals the figure it exists to withhold — somebody's exact
MRR plus a public plan price is their customer count.

`ok` is false for a currency the dollar ladder does not cover, and for no
revenue at all. **Do not render that as the bottom band** — "no revenue
verified" and "under $100" are different claims.

{{< callout type="warning" >}}
If you publish a leaderboard, order by band and then by who verified first,
never by amount. Ordering within a band publishes an ordering finer than the
band.
{{< /callout >}}

## Testing

```go
gateway := paytest.New("whsec_test")
payload := gateway.Event("evt_1", "customer.subscription.updated", now, sub, "user_42")
signature := gateway.Sign(payload, now)

err := billing.Handle(ctx, payload, signature)
```

No network is involved. `paytest.Gateway` verifies with the same code the real
driver uses, so a test passing against the fake exercises the real signature
scheme, and it records `Checkouts()` and `Portals()` for asserting on what
would have been sent.

## Limitations

No refunds, invoices, metered usage, tax, one-off charges, or a cancel flow of
its own — the gateway's hosted pages cover those, and wrapping them here would
make the interface Stripe-shaped.

{{< callout type="info" >}}
**Read prices from the gateway rather than mirroring them in environment
variables.** Two copies of a price eventually disagree, and then the one your
checkout charges is not the one your pricing page shows.
{{< /callout >}}

## Next steps

- [Session](/docs/session) — where entitlement is usually read from.
- [Authorization](/docs/authz/) — gating a feature on an active subscription.
