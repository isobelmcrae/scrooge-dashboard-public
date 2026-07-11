package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// zeroDecimal are currencies Stripe stores with no minor unit, where the integer
// amount already is the whole value, so we must not divide by 100.
// ponytail: only the common ones; add more from stripe.com/docs/currencies if needed.
var zeroDecimal = map[string]bool{
	"BIF": true, "CLP": true, "DJF": true, "GNF": true, "JPY": true, "KMF": true,
	"KRW": true, "MGA": true, "PYG": true, "RWF": true, "UGX": true, "VND": true,
	"VUV": true, "XAF": true, "XOF": true, "XPF": true,
}

var symbols = map[string]string{
	"USD": "$", "AUD": "A$", "EUR": "€", "GBP": "£", "CAD": "C$",
	"NZD": "NZ$", "JPY": "¥", "SGD": "S$",
}

// money renders minor units in a currency as a human string, e.g. "$1,299.00".
func money(minor int64, currency string) string {
	currency = strings.ToUpper(currency)
	sym := symbols[currency]

	var body string
	if zeroDecimal[currency] {
		body = withCommas(minor)
	} else {
		whole := minor / 100
		cents := minor % 100
		if cents < 0 {
			cents = -cents
		}
		body = withCommas(whole) + "." + fmt.Sprintf("%02d", cents)
	}
	if sym != "" {
		return sym + body
	}
	return body + " " + currency
}

// minorToMajor converts minor units to a major-unit float (cents to dollars),
// leaving zero-decimal currencies untouched.
func minorToMajor(minor int64, currency string) float64 {
	if zeroDecimal[strings.ToUpper(currency)] {
		return float64(minor)
	}
	return float64(minor) / 100
}

// majorToMinor is the inverse of minorToMajor, rounded to the nearest unit.
func majorToMinor(major float64, currency string) int64 {
	if zeroDecimal[strings.ToUpper(currency)] {
		return int64(math.Round(major))
	}
	return int64(math.Round(major * 100))
}

func withCommas(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	// insert thousands separators
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}
