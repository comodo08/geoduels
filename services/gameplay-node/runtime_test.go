package main

import (
	"testing"

	"geoduels/pkg/contracts"
	"geoduels/pkg/singleplayer"
)

func TestRoundPlanRegistryReturnsPinnedRounds(t *testing.T) {
	registry := newRoundPlanRegistry()
	registry.Set("match", []contracts.PlannedRound{
		{RoundIndex: 0, Location: contracts.LocationPoint{Lat: 1, Lng: 2}},
		{RoundIndex: 1, Location: contracts.LocationPoint{Lat: 3, Lng: 4}},
	}, nil)
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

func TestRoundPlanRegistryExtenderGeneratesBeyondPlan(t *testing.T) {
	registry := newRoundPlanRegistry()
	registry.Set("match", []contracts.PlannedRound{
		{RoundIndex: 0, Location: contracts.LocationPoint{Lat: 1, Lng: 2}},
		{RoundIndex: 1, Location: contracts.LocationPoint{Lat: 3, Lng: 4}},
	}, func(roundIndex int) (contracts.LocationPoint, error) {
		return contracts.LocationPoint{Lat: float64(roundIndex), Lng: float64(roundIndex)}, nil
	})

	point, err := registry.Get("match", 5)
	if err != nil {
		t.Fatalf("expected extender to generate round 5: %v", err)
	}
	if point.Lat != 5 || point.Lng != 5 {
		t.Fatalf("unexpected extended point %+v", point)
	}
	if len(registry.plans["match"]) != 2 {
		t.Fatalf("expected plan to stay pinned at 2 rounds, got %d", len(registry.plans["match"]))
	}
	again, err := registry.Get("match", 5)
	if err != nil {
		t.Fatalf("expected second generation to succeed: %v", err)
	}
	if again != point {
		t.Fatalf("expected deterministic generation, got %+v then %+v", point, again)
	}

	if _, err := registry.Get("no-extender", 3); err == nil {
		t.Fatal("expected exhausted plan error for match without extender")
	}
}

func TestSingleplayerRuntimePreservesMatchConfig(t *testing.T) {
	runtime := singleplayerRuntime{
		engine: singleplayer.New(func(matchID string, roundIndex int) (contracts.LocationPoint, error) {
			return contracts.LocationPoint{Lat: 1, Lng: 2, Country: "US"}, nil
		}),
	}
	err := runtime.CreateMatch(
		"solo-config",
		[]string{"u1"},
		map[string]contracts.PlayerProfile{
			"u1": {UserID: "u1", DisplayName: "Solo"},
		},
		false,
		"",
		contracts.MatchConfig{
			Ruleset:     contracts.RulesetNoMove,
			StreetNames: contracts.StreetNamesHidden,
		},
		nil,
	)
	if err != nil {
		t.Fatalf("create match: %v", err)
	}

	snap, err := runtime.GetSnapshot("solo-config")
	if err != nil {
		t.Fatalf("get snapshot: %v", err)
	}
	if snap.Config.Ruleset != contracts.RulesetNoMove {
		t.Fatalf("ruleset = %q, want %q", snap.Config.Ruleset, contracts.RulesetNoMove)
	}
	if snap.Config.StreetNames != contracts.StreetNamesHidden {
		t.Fatalf("street names = %q, want %q", snap.Config.StreetNames, contracts.StreetNamesHidden)
	}
}
