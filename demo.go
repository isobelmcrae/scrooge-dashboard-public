package main

import "time"

// demoCharges fabricates a realistic month of inflows so the dashboard is
// viewable without a Stripe key: USD + AUD, renewals, a trial conversion, and a
// couple of at-risk retries. Anchored to `now` so it always fills the view.
func demoCharges(now time.Time) []Charge {
	loc := now.Location()
	y, m, _ := now.Date()
	first := time.Date(y, m, 1, 12, 0, 0, 0, loc)
	on := func(day int) time.Time { return time.Date(y, m, day, 12, 0, 0, 0, loc) }

	c := []Charge{
		{on(3), 4900, "USD", KindRenewal, "Marley & Co"},
		{on(3), 2900, "USD", KindRenewal, "Cratchit Ltd"},
		{on(5), 12000, "AUD", KindRenewal, "Fezziwig Pty"},
		{on(7), 990, "USD", KindTrial, "Tiny Tim (trial ends)"},
		{on(9), 4900, "USD", KindRenewal, "Scrooge Holdings"},
		{on(9), 8500, "AUD", KindRenewal, "Bob's Coal Supply"},
		{on(12), 29900, "USD", KindRenewal, "Humbug Enterprises"},
		{on(14), 4900, "USD", KindRetry, "Past-due Partners"},
		{on(15), 15000, "AUD", KindRenewal, "Marley & Co (AU)"},
		{on(18), 990, "USD", KindRenewal, "Ghost of Christmas Past"},
		{on(18), 990, "USD", KindRenewal, "Ghost of Christmas Present"},
		{on(18), 990, "USD", KindRenewal, "Ghost of Christmas Future"},
		{on(21), 49900, "USD", KindRenewal, "Dickens & Sons"},
		{on(22), 6500, "AUD", KindRetry, "Overdue Outback Co"},
		{on(24), 4900, "USD", KindRenewal, "Belle Financial"},
		{on(26), 19900, "AUD", KindRenewal, "Antipodean Alloys"},
		{on(28), 990, "USD", KindTrial, "Fred's Feast (trial ends)"},
		{on(28), 4900, "USD", KindRenewal, "Jacob Marley Trust"},
	}
	_ = first
	return c
}
