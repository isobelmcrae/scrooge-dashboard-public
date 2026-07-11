package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// fxCache holds the latest exchange rates for one base currency, refreshed at
// most hourly. ponytail: one process-wide cache, no store; rates barely move.
type fxCache struct {
	mu      sync.Mutex
	base    string
	rates   map[string]float64 // units of currency per 1 base
	fetched time.Time
}

var fx = &fxCache{}

// fxRates returns exchange rates (currency per 1 base) from open.er-api.com,
// cached for an hour. No API key required.
func fxRates(ctx context.Context, base string) (map[string]float64, error) {
	base = strings.ToUpper(base)

	fx.mu.Lock()
	defer fx.mu.Unlock()
	if fx.base == base && fx.rates != nil && time.Since(fx.fetched) < time.Hour {
		return fx.rates, nil
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://open.er-api.com/v6/latest/"+base, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fx: status %d", resp.StatusCode)
	}

	var out struct {
		Result string             `json:"result"`
		Rates  map[string]float64 `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.Result != "success" || len(out.Rates) == 0 {
		return nil, fmt.Errorf("fx: bad response")
	}
	out.Rates[base] = 1 // base to itself

	fx.base, fx.rates, fx.fetched = base, out.Rates, time.Now()
	return out.Rates, nil
}

// convertCharges restates every charge into the base currency using rates.
// Charges in a currency missing from rates are dropped (can't price them).
func convertCharges(charges []Charge, base string, rates map[string]float64) []Charge {
	base = strings.ToUpper(base)
	out := make([]Charge, 0, len(charges))
	for _, c := range charges {
		amt, ok := convertMinor(c.Amount, c.Currency, base, rates)
		if !ok {
			continue
		}
		c.Amount = amt
		c.Currency = base
		out = append(out, c)
	}
	return out
}

// convertMinor converts minor units from one currency to another via rates
// (units per 1 base). Returns ok=false if the source rate is unknown.
func convertMinor(minor int64, from, to string, rates map[string]float64) (int64, bool) {
	from, to = strings.ToUpper(from), strings.ToUpper(to)
	if from == to {
		return minor, true
	}
	r, ok := rates[from]
	if !ok || r == 0 {
		return 0, false
	}
	// rates are "from-currency per 1 base(to)", so divide to reach the base.
	return majorToMinor(minorToMajor(minor, from)/r, to), true
}
