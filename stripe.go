package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	stripe "github.com/stripe/stripe-go/v86"
)

// FetchCharges pulls active/trialing subscriptions and open (retrying) invoices
// from Stripe and projects every expected charge that lands in [from, to].
// It is read-only.
func FetchCharges(ctx context.Context, sc *stripe.Client, from, to time.Time, pessimistic bool) ([]Charge, error) {
	var out []Charge

	// --- Subscriptions: project recurring renewals + trial first-charges ---
	subParams := &stripe.SubscriptionListParams{}
	subParams.Limit = stripe.Int64(100)
	subParams.AddExpand("data.items.data.price")
	subParams.AddExpand("data.customer")
	// No status filter → all non-canceled subs; we switch on status below.

	for sub, err := range sc.V1Subscriptions.List(ctx, subParams).All(ctx) {
		if err != nil {
			return nil, fmt.Errorf("listing subscriptions: %w", err)
		}
		if pessimistic && sub.Status != stripe.SubscriptionStatusActive {
			continue
		}
		out = append(out, projectSubscription(sub, from, to)...)
	}
	if pessimistic {
		return out, nil
	}

	// --- Open invoices: dunning retries are at-risk money on their retry date ---
	invParams := &stripe.InvoiceListParams{Status: stripe.String("open")}
	invParams.Limit = stripe.Int64(100)
	invParams.AddExpand("data.customer")

	for inv, err := range sc.V1Invoices.List(ctx, invParams).All(ctx) {
		if err != nil {
			return nil, fmt.Errorf("listing open invoices: %w", err)
		}
		if inv.NextPaymentAttempt == 0 || inv.AmountDue == 0 {
			continue // auto-retries exhausted, or nothing owed
		}
		t := time.Unix(inv.NextPaymentAttempt, 0)
		if t.Before(from) || t.After(to) {
			continue
		}
		out = append(out, Charge{
			Date:     t,
			Amount:   inv.AmountDue,
			Currency: strings.ToUpper(string(inv.Currency)),
			Kind:     KindRetry,
			Who:      customerName(inv.Customer),
		})
	}

	return out, nil
}

// projectSubscription emits one Charge per billing cycle of sub that falls in
// [from, to]. Trial subs bill first at trial_end, then on the normal interval.
func projectSubscription(sub *stripe.Subscription, from, to time.Time) []Charge {
	switch sub.Status {
	case stripe.SubscriptionStatusActive, stripe.SubscriptionStatusTrialing:
		// project these
	default:
		// past_due / unpaid money shows up as open-invoice retries instead;
		// incomplete / paused / canceled won't bill on schedule.
		return nil
	}
	if sub.CancelAtPeriodEnd {
		return nil // ends this period, no future renewal
	}
	if sub.Items == nil || len(sub.Items.Data) == 0 {
		return nil
	}

	// Sum price across items (a subscription is single-currency).
	var amount int64
	var currency string
	var rec *stripe.PriceRecurring
	for _, it := range sub.Items.Data {
		if it.Price == nil {
			continue
		}
		qty := it.Quantity
		if qty < 1 {
			qty = 1
		}
		amount += it.Price.UnitAmount * qty
		currency = strings.ToUpper(string(it.Price.Currency))
		if rec == nil {
			rec = it.Price.Recurring
		}
	}
	if amount == 0 || currency == "" || rec == nil {
		return nil
	}

	// First expected charge date + kind.
	start := time.Unix(sub.Items.Data[0].CurrentPeriodEnd, 0)
	kind := KindRenewal
	if sub.Status == stripe.SubscriptionStatusTrialing && sub.TrialEnd > 0 {
		start = time.Unix(sub.TrialEnd, 0)
		kind = KindTrial
	}

	who := customerName(sub.Customer)

	var out []Charge
	// ponytail: for daily plans viewed far out this loops per-day; bounded by the
	// window (~6 weeks) so it stays cheap. Cap iterations as a runaway guard.
	for t, n := start, 0; !t.After(to) && n < 500; t, n = addInterval(t, rec), n+1 {
		if t.Before(from) {
			kind = KindRenewal // any charge after the first is a plain renewal
			continue
		}
		out = append(out, Charge{
			Date:     t,
			Amount:   amount,
			Currency: currency,
			Kind:     kind,
			Who:      who,
		})
		kind = KindRenewal
	}
	return out
}

func addInterval(t time.Time, rec *stripe.PriceRecurring) time.Time {
	n := int(rec.IntervalCount)
	if n < 1 {
		n = 1
	}
	switch rec.Interval {
	case stripe.PriceRecurringIntervalDay:
		return t.AddDate(0, 0, n)
	case stripe.PriceRecurringIntervalWeek:
		return t.AddDate(0, 0, 7*n)
	case stripe.PriceRecurringIntervalMonth:
		return t.AddDate(0, n, 0)
	case stripe.PriceRecurringIntervalYear:
		return t.AddDate(n, 0, 0)
	default:
		return t.AddDate(0, 1, 0)
	}
}

func customerName(c *stripe.Customer) string {
	if c == nil {
		return "unknown"
	}
	switch {
	case c.Name != "":
		return c.Name
	case c.Email != "":
		return c.Email
	default:
		return c.ID
	}
}
