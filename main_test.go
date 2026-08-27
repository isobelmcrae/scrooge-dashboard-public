package main

import "testing"

func TestOnlyRenewals(t *testing.T) {
	charges := []Charge{
		{Kind: KindRenewal},
		{Kind: KindTrial},
		{Kind: KindRetry},
	}

	got := onlyRenewals(charges)
	if len(got) != 1 || got[0].Kind != KindRenewal {
		t.Fatalf("onlyRenewals() = %#v, want one renewal", got)
	}
}
