package main

import "time"

// Kind is why money lands on a day.
type Kind string

const (
	KindRenewal Kind = "renewal" // a recurring subscription bills
	KindTrial   Kind = "trial"   // a trial ends and bills for the first time
	KindRetry   Kind = "retry"   // a failed invoice is scheduled to retry (at-risk)
)

// Charge is one expected inflow on a given day, in a single currency.
type Charge struct {
	Date     time.Time
	Amount   int64  // minor units (cents for USD/AUD)
	Currency string // upper-case ISO code, e.g. "USD"
	Kind     Kind
	Who      string // customer name / email / id
}

// AtRisk reports whether this money is a dunning retry rather than a clean charge.
func (c Charge) AtRisk() bool { return c.Kind == KindRetry }
