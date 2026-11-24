## When to use this skill

When taking a payment or running a subscription. One hosted checkout, one
signed webhook, Stripe driver over `net/http`.

The expensive mistakes in billing are not in the API calls — they are in what
the application believes afterwards.

## Entitlement comes from the webhook, never from the redirect

A browser returning to a success URL proves only that a browser was sent there:
the person can close the tab, the redirect can be lost, and the URL can be
visited again tomorrow by anyone who remembers it.

The webhook is signed, retried until acknowledged, and is the only thing
`Handle` will write from. **Do not grant access on the success page.** Render
"we are setting up your account" and let the webhook do the write.

## A delivery arrives more than once, and out of order

Stripe retries until it gets a 2xx and promises no ordering at all.

- `Store.Seen` deduplicates on the event id.
- `Store.Apply` must let an older event lose against a newer one — **as a
  condition on the write, not a read followed by a write.** A read-then-write
  guard has a window exactly wide enough to lose the newer event.

So the `Apply` implementation is a single `UPDATE ... WHERE updated_at <
$incoming` or equivalent, never a `SELECT` then an `UPDATE`.

## The signature is checked over the exact bytes received

With the timestamp inside the MAC and a tolerance around it
(`DefaultTolerance`, 5 minutes). Two consequences:

- **Keep the raw body.** Decoding and re-encoding the same JSON changes the
  signature, so a handler that binds to a struct first and verifies second
  passes only by luck. `MaxWebhookBody` is 1 MB.
- Without the tolerance a captured signature would be valid forever.

Two webhook secrets are supported so one can be rotated without an interval in
which events are refused — and a refused event is money moving with nothing
recording it.

## Money is an integer, always

Amounts are minor units — cents — in `int64`. `Factor(currency)` gives the
multiplier, because not every currency has two decimal places. `ErrMixedCurrency`
is returned rather than summing across currencies.

Never introduce a float for money anywhere on this path.

## Status and the grace period

A subscription past due stays usable for `GracePeriod` (48 hours) rather than
cutting off at the instant a card fails, because the common cause is an expiry
the customer will fix. Read the status rather than comparing dates yourself.

## `Store` is yours to implement

The module does not own your tables. Verified revenue is a **separate
interface** from entitlement, so a reporting query cannot accidentally be
answered from the same optimistic state a feature gate uses.

## Check first

`search-docs` section `payments`, `module-api` on `pay`. `ErrNoGateway` means
nothing is configured.
