package main

import (
	"sort"
	"time"
)

// Amount pairs a currency with a total, for ordered display.
type Amount struct {
	Currency string
	Minor    int64
}

// Day is one cell in the calendar grid.
type Day struct {
	Date     time.Time
	InMonth  bool
	Today    bool
	Past     bool
	Expected []Amount // clean incoming money (renewals + trials), per currency
	Risk     []Amount // dunning retries, per currency
	Charges  []Charge // everything landing this day, sorted
	Glow     float64  // 0..1 visual intensity relative to the month's biggest day
}

// HasMoney reports whether anything at all lands on this day.
func (d Day) HasMoney() bool { return len(d.Charges) > 0 }

// Month is the full view model handed to the template.
type Month struct {
	Label    string // "July 2026"
	Key      string // "2026-07"
	PrevKey  string
	NextKey  string
	Weeks    [][]Day
	Weekdays []string

	Expected    []Amount // month totals, clean money, per currency
	Risk        []Amount // month totals, at-risk money, per currency
	Biggest     *Day     // busiest single day
	ChargeCount int
	Ledger      []Charge // every upcoming charge this month, date-sorted
}

// BuildMonth aggregates charges into a Monday-start calendar grid for the month
// containing `anchor`. `now` marks today for past/greying.
func BuildMonth(anchor, now time.Time, charges []Charge) Month {
	loc := anchor.Location()
	year, mon, _ := anchor.Date()
	first := time.Date(year, mon, 1, 0, 0, 0, 0, loc)
	next := first.AddDate(0, 1, 0)

	// Bucket charges by day.
	byDay := map[string][]Charge{}
	for _, c := range charges {
		byDay[dayKey(c.Date)] = append(byDay[dayKey(c.Date)], c)
	}

	// Grid: back up to Monday, run full weeks until the month is covered.
	start := first
	for start.Weekday() != time.Monday {
		start = start.AddDate(0, 0, -1)
	}
	today := dayKey(now)

	var weeks [][]Day
	var biggest *Day
	expected := map[string]int64{}
	risk := map[string]int64{}
	var ledger []Charge
	count := 0

	// First pass: find the biggest day magnitude for glow scaling.
	var maxMag int64
	for _, cs := range byDay {
		var mag int64
		for _, c := range cs {
			mag += c.Amount // rough cross-currency magnitude, visual only
		}
		if mag > maxMag {
			maxMag = mag
		}
	}

	for d := start; d.Before(next) || d.Weekday() != time.Monday; d = d.AddDate(0, 0, 1) {
		key := dayKey(d)
		cs := byDay[key]
		sortCharges(cs)

		day := Day{
			Date:    d,
			InMonth: !d.Before(first) && d.Before(next),
			Today:   key == today,
			Past:    dayKey(d) < today,
			Charges: cs,
		}

		if day.InMonth && len(cs) > 0 {
			exp := map[string]int64{}
			rsk := map[string]int64{}
			var mag int64
			for _, c := range cs {
				mag += c.Amount
				if c.AtRisk() {
					rsk[c.Currency] += c.Amount
					risk[c.Currency] += c.Amount
				} else {
					exp[c.Currency] += c.Amount
					expected[c.Currency] += c.Amount
				}
				ledger = append(ledger, c)
				count++
			}
			day.Expected = toAmounts(exp)
			day.Risk = toAmounts(rsk)
			if maxMag > 0 {
				day.Glow = float64(mag) / float64(maxMag)
			}
		}

		// Track busiest day (by magnitude).
		if day.InMonth && day.HasMoney() {
			if biggest == nil || dayMag(*biggest) < dayMag(day) {
				d := day
				biggest = &d
			}
		}

		week := len(weeks) - 1
		if week < 0 || len(weeks[week]) == 7 {
			weeks = append(weeks, nil)
			week++
		}
		weeks[week] = append(weeks[week], day)
	}

	sortCharges(ledger)

	return Month{
		Label:       first.Format("January 2006"),
		Key:         first.Format("2006-01"),
		PrevKey:     first.AddDate(0, -1, 0).Format("2006-01"),
		NextKey:     next.Format("2006-01"),
		Weeks:       weeks,
		Weekdays:    []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
		Expected:    toAmounts(expected),
		Risk:        toAmounts(risk),
		Biggest:     biggest,
		ChargeCount: count,
		Ledger:      ledger,
	}
}

func dayMag(d Day) int64 {
	var m int64
	for _, c := range d.Charges {
		m += c.Amount
	}
	return m
}

func dayKey(t time.Time) string { return t.Format("2006-01-02") }

// toAmounts turns a currency→total map into a currency-sorted slice.
func toAmounts(m map[string]int64) []Amount {
	out := make([]Amount, 0, len(m))
	for cur, v := range m {
		out = append(out, Amount{Currency: cur, Minor: v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Currency < out[j].Currency })
	return out
}

// sortCharges orders by date, then at-risk last, then amount desc.
func sortCharges(cs []Charge) {
	sort.SliceStable(cs, func(i, j int) bool {
		if !cs[i].Date.Equal(cs[j].Date) {
			return cs[i].Date.Before(cs[j].Date)
		}
		if cs[i].AtRisk() != cs[j].AtRisk() {
			return !cs[i].AtRisk()
		}
		return cs[i].Amount > cs[j].Amount
	})
}
