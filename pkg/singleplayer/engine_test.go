package singleplayer

import (
	"testing"

	"geoduels/pkg/contracts"
)

func TestSingleplayerFlowFinalizeAndAdvance(t *testing.T) {
	engine := New(func(matchID string, roundIndex int) (contracts.LocationPoint, error) {
		return contracts.LocationPoint{
			Lat:     float64(roundIndex),
			Lng:     float64(roundIndex),
			Country: "US",
		}, nil
	})
	_, err := engine.CreateMatch("solo-1", []string{"u1"}, map[string]contracts.PlayerProfile{
		"u1": {UserID: "u1", DisplayName: "Solo"},
	})
	if err != nil {
		t.Fatalf("create match: %v", err)
	}

	snap, err := engine.SubmitGuess(contracts.GuessPayload{
		UserID:   "u1",
		MatchID:  "solo-1",
		RoundID:  "solo-1:r1",
		Lat:      0,
		Lng:      0,
		Finalize: true,
	})
	if err != nil {
		t.Fatalf("submit guess: %v", err)
	}
	if snap.Mode != contracts.ModeSingleplayer {
		t.Fatalf("expected singleplayer mode, got %q", snap.Mode)
	}
	if snap.Phase != contracts.PhaseRoundResult {
		t.Fatalf("expected round result phase, got %q", snap.Phase)
	}
	if snap.LastRoundResult == nil {
		t.Fatal("expected round result")
	}
	if len(snap.RoundResults) != 1 {
		t.Fatalf("expected round history length 1, got %d", len(snap.RoundResults))
	}
	if snap.Players["u1"].TotalScore <= 0 {
		t.Fatalf("expected score to accumulate, got %d", snap.Players["u1"].TotalScore)
	}

	snap, err = engine.AdvanceRound("solo-1", "u1")
	if err != nil {
		t.Fatalf("advance round: %v", err)
	}
	if snap.Phase != contracts.PhaseLive {
		t.Fatalf("expected live phase after advance, got %q", snap.Phase)
	}
	if snap.CurrentRound == nil || snap.CurrentRound.RoundNumber != 2 {
		t.Fatalf("expected round 2, got %+v", snap.CurrentRound)
	}
	if len(snap.RoundResults) != 1 {
		t.Fatalf("expected round history to persist after advance, got %d", len(snap.RoundResults))
	}
}

func TestSingleplayerSnapshotPreservesMatchConfig(t *testing.T) {
	engine := New(func(matchID string, roundIndex int) (contracts.LocationPoint, error) {
		return contracts.LocationPoint{
			Lat:     float64(roundIndex),
			Lng:     float64(roundIndex),
			Country: "US",
		}, nil
	})
	_, err := engine.CreateMatchWithConfig("solo-config", []string{"u1"}, map[string]contracts.PlayerProfile{
		"u1": {UserID: "u1", DisplayName: "Solo"},
	}, contracts.MatchConfig{
		Ruleset:     contracts.RulesetNoMove,
		StreetNames: contracts.StreetNamesHidden,
	})
	if err != nil {
		t.Fatalf("create match: %v", err)
	}

	snap, err := engine.GetSnapshot("solo-config")
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

func TestSingleplayerEndlessAdvancesPastMaxRounds(t *testing.T) {
	engine := New(func(matchID string, roundIndex int) (contracts.LocationPoint, error) {
		return contracts.LocationPoint{
			Lat:     float64(roundIndex),
			Lng:     float64(roundIndex),
			Country: "US",
		}, nil
	})
	_, err := engine.CreateMatchWithConfig("solo-endless", []string{"u1"}, map[string]contracts.PlayerProfile{
		"u1": {UserID: "u1", DisplayName: "Solo"},
	}, contracts.MatchConfig{
		Endless: true,
	})
	if err != nil {
		t.Fatalf("create match: %v", err)
	}

	for i := 0; i < maxRounds+5; i++ {
		if _, err := engine.SubmitGuess(contracts.GuessPayload{
			UserID:   "u1",
			MatchID:  "solo-endless",
			RoundID:  roundID("solo-endless", i+1),
			Lat:      0,
			Lng:      0,
			Finalize: true,
		}); err != nil {
			t.Fatalf("submit guess round %d: %v", i+1, err)
		}
		snap, err := engine.AdvanceRound("solo-endless", "u1")
		if err != nil {
			t.Fatalf("advance round %d: %v", i+1, err)
		}
		if snap.State != contracts.MatchLive {
			t.Fatalf("expected endless match to stay live past round %d, got state %q", i+1, snap.State)
		}
		if len(snap.RoundResults) != i+1 {
			t.Fatalf("expected %d round results, got %d", i+1, len(snap.RoundResults))
		}
	}

	snap, err := engine.GetSnapshot("solo-endless")
	if err != nil {
		t.Fatalf("get snapshot: %v", err)
	}
	if snap.Config.Endless != true {
		t.Fatalf("expected endless config to be preserved, got %v", snap.Config.Endless)
	}
}

func TestSingleplayerNonEndlessStopsAtMaxRounds(t *testing.T) {
	engine := New(func(matchID string, roundIndex int) (contracts.LocationPoint, error) {
		return contracts.LocationPoint{
			Lat:     float64(roundIndex),
			Lng:     float64(roundIndex),
			Country: "US",
		}, nil
	})
	_, err := engine.CreateMatch("solo-capped", []string{"u1"}, map[string]contracts.PlayerProfile{
		"u1": {UserID: "u1", DisplayName: "Solo"},
	})
	if err != nil {
		t.Fatalf("create match: %v", err)
	}

	for i := 0; i < maxRounds; i++ {
		if _, err := engine.SubmitGuess(contracts.GuessPayload{
			UserID:   "u1",
			MatchID:  "solo-capped",
			RoundID:  roundID("solo-capped", i+1),
			Lat:      0,
			Lng:      0,
			Finalize: true,
		}); err != nil {
			t.Fatalf("submit guess round %d: %v", i+1, err)
		}
		snap, err := engine.AdvanceRound("solo-capped", "u1")
		if err != nil {
			t.Fatalf("advance round %d: %v", i+1, err)
		}
		if i < maxRounds-1 {
			if snap.State != contracts.MatchLive {
				t.Fatalf("expected live at round %d, got %q", i+1, snap.State)
			}
		} else if snap.State != contracts.MatchEnded {
			t.Fatalf("expected match ended after %d rounds, got %q", maxRounds, snap.State)
		}
	}
}

func TestSingleplayerDisconnectResume(t *testing.T) {
	engine := New(func(matchID string, roundIndex int) (contracts.LocationPoint, error) {
		return contracts.LocationPoint{
			Lat:     float64(roundIndex),
			Lng:     float64(roundIndex),
			Country: "US",
		}, nil
	})
	_, err := engine.CreateMatch("solo-disconnect", []string{"u1"}, map[string]contracts.PlayerProfile{
		"u1": {UserID: "u1", DisplayName: "Solo"},
	})
	if err != nil {
		t.Fatalf("create match: %v", err)
	}

	snap, err := engine.MarkDisconnected("solo-disconnect", "u1")
	if err != nil {
		t.Fatalf("disconnect: %v", err)
	}
	if !snap.Players["u1"].Disconnected {
		t.Fatal("expected player to be disconnected")
	}

	snap, err = engine.MarkResumed("solo-disconnect", "u1")
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if snap.Players["u1"].Disconnected {
		t.Fatal("expected player to be resumed")
	}
}
