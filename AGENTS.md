# AGENTS.md

Guidance for AI coding agents working in this repo. Read the `README.md` first for what the app does and how it is structured.

## Ethos

This repo was built ponytail-style: the laziest solution that works. Prefer the standard library, keep the diff smallest, and add no speculative abstractions. No interface with one implementation, no config for a value that never changes. If a change does not need to exist, do not make it.

## Ground rules

- Keep it read-only. Never add Stripe write calls. A read-only restricted key is a feature, not a limitation.
- Only active and trialing subscriptions are projected forward. Money for `past_due` / `unpaid` subscriptions appears via open invoices (retries) instead, to avoid double counting. Subscriptions with `cancel_at_period_end` set are skipped.
- Currencies are never summed across each other and there is no FX conversion.

## Gotchas

- stripe-go v86 moved the billing period OFF the `Subscription` onto each subscription item. Use `sub.Items.Data[i].CurrentPeriodEnd`. The old `subscription.current_period_end` is gone. The renewal amount is summed from the item prices (`UnitAmount * Quantity`).
- Assets are embedded with `//go:embed` at compile time. Editing `templates/` or `static/` has no effect until you rebuild and restart.
- Zero-decimal currencies (JPY and similar) must not be divided by 100. `money.go` guards this. Do not bypass it.

## Run and verify locally

```sh
go build ./...
go vet ./...
SCROOGE_DEMO=1 go run .
# then load http://localhost:8080 and confirm the calendar renders
```

Demo mode needs no Stripe key, so it is the fastest way to check that a change renders correctly.
