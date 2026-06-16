package main

import (
	"testing"

	"geoduels/pkg/contracts"
)

func TestRoundPlanRegistryReturnsPinnedRounds(t *testing.T) {
	registry := newRoundPlanRegistry()
	registry.Set("match", []contracts.PlannedRound{
		{RoundIndex: 0, Location: contracts.LocationPoint{Lat: 1, Lng: 2}},
		{RoundIndex: 1, Location: contracts.LocationPoint{Lat: 3, Lng: 4}},
	})
	point, err := registry.Get("match", 1)
	if err != nil {
		t.Fatal(err)
	}
	if point.Lat != 3 || point.Lng != 4 {
		t.Fatalf("unexpected point %+v", point)
	}
	if _, err := registry.Get("match", 2); err == nil {
		t.Fatal("expected exhausted plan error")
	}
}
