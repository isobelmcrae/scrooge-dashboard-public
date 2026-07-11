# Scrooge Dashboard

A tiny read-only web app that connects to one Stripe account and shows a calendar of upcoming money: how much revenue will land on each upcoming day of the month.

It accounts for:

- Recurring subscription renewals
- Trial subscriptions (first charge at trial end)
- Failed payments that are retrying (dunning)

Money is bucketed by day and by currency. Multiple currencies are supported and shown separately per currency. USD and AUD are never summed together, and there is no FX conversion.

The dates shown are charge / renewal dates (when Stripe bills the customer), not bank payout settlement dates. Funds settle a couple of days later, and the app notes this.

## Stack

- Go, standard library `net/http` + `html/template`, compiled to a single binary.
- Stripe Go SDK: `github.com/stripe/stripe-go/v86`.
- Templates and static assets are embedded with `//go:embed`, so the binary is self-contained (no external files needed at runtime). Because assets are embedded at compile time, you must rebuild and restart after editing `templates/` or `static/`.
- No JavaScript framework, no database. The frontend is server-rendered HTML plus one CSS file.

## Running

```sh
go run .
# then open http://localhost:8080
```

Or build a binary:

```sh
go build -o scrooge . && ./scrooge
```

Navigate months with the arrows in the UI. The URL is `/?m=YYYY-MM`.

## Configuration

All configuration is via environment variables.

| Variable | Purpose |
|----------|---------|
| `STRIPE_API_KEY` | Stripe secret or restricted key. If unset, the app runs in demo mode with sample data. |
| `SCROOGE_DEMO` | Set to `1` to force demo mode even when a key is set. |
| `SCROOGE_ADDR` | Listen address. Defaults to `:8080`. |
| `SCROOGE_BASE_CURRENCY` | Currency the "converted" view rolls everything into. Defaults to `USD`. |

## Currency views

The header has a toggle:

- **Per currency** (default): each currency is shown and totalled on its own. USD and AUD are never mixed. No exchange rates involved.
- **In USD** (or whatever `SCROOGE_BASE_CURRENCY` is set to): everything is converted into one currency so you get a single bottom line.

Conversion uses daily reference rates from `open.er-api.com` (no API key, cached for an hour in memory). If those rates cannot be fetched, the app falls back to the per-currency view and says so. The toggle state is carried in the URL (`?view=split` or `?view=convert`).

## Stripe API key

Use a restricted key (it starts with `rk_`), created in the Stripe Dashboard under Developers > API keys > Create restricted key.

Grant read permission (not write) to exactly these resources:

- Subscriptions: read
- Invoices: read
- Customers: read

Everything else can stay "None".

The app never writes to Stripe, so a read-only restricted key is the safe choice. Do not use the account's full secret key. The key works with both test-mode and live-mode keys, and test mode is a safe way to try it.

Pass the key via the `STRIPE_API_KEY` environment variable and never commit it:

```sh
STRIPE_API_KEY=rk_live_xxx go run .
```

## Architecture

| File | Responsibility |
|------|----------------|
| `main.go` | HTTP server, request handler, month selection (`?m=YYYY-MM`), demo-mode wiring, template execution. |
| `stripe.go` | Read-only Stripe fetching. Lists active / trialing subscriptions and projects each billing cycle forward across the visible window, and lists open (retrying) invoices via `next_payment_attempt`. Returns a flat list of `Charge` values. |
| `calendar.go` | Aggregates charges into a Monday-start month grid (the view model), and computes per-currency month totals, biggest day, and the sorted "ledger" list. |
| `charge.go` | The `Charge` type and its `Kind` (renewal / trial / retry). |
| `money.go` | Converts minor units to a human currency string, with a zero-decimal-currency guard (JPY and similar are not divided by 100, USD and AUD are). |
| `demo.go` | Fabricated sample data so the UI is viewable without a Stripe key. |
| `templates/index.html`, `static/style.css` | The single page. |
