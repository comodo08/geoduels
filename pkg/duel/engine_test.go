package duel

import (
	"testing"
	"time"

	"geoduels/pkg/contracts"
)

func TestCreateAndGuessFlow(t *testing.T) {
	rounds := []contracts.LocationPoint{
		{Lat: 10, Lng: 10, Country: "US"},
		{Lat: 20, Lng: 20, Country: "US"},
	}
	e := New(func(_ string, i int) (contracts.LocationPoint, error) { return rounds[i%len(rounds)], nil })
	m, err := e.CreateMatch("m1", []string{"u1", "u2"}, nil)
	if err != nil {
		t.Fatalf("create match: %v", err)
	}
	m.RoundStartedAt = time.Now().Add(-roundIntro)
	snap, err := e.GetSnapshot("m1")
	if err != nil {
		t.Fatal(err)
	}
	if snap.CurrentRound == nil || snap.CurrentRound.RoundNumber != 1 {
		t.Fatalf("expected round 1 snapshot")
	}

	_, err = e.SubmitGuess(contracts.GuessPayload{UserID: "u1", MatchID: "m1", RoundID: snap.CurrentRound.RoundID, Lat: 10.1, Lng: 10.1, Finalize: true})
	if err != nil {
		t.Fatal(err)
	}
	snap2, err := e.SubmitGuess(contracts.GuessPayload{UserID: "u2", MatchID: "m1", RoundID: snap.CurrentRound.RoundID, Lat: 55.0, Lng: 40.0, Finalize: true})
	if err != nil {
		t.Fatal(err)
	}
	if snap2.State != contracts.MatchLive && snap2.State != contracts.MatchEnded {
		t.Fatalf("invalid state: %s", snap2.State)
	}
	if snap2.Players["u2"].HP >= 5000 {
		t.Fatalf("expected u2 hp to drop")
	}
}

func TestCreateMatchIncludesRatingPreview(t *testing.T) {
	rounds := []contracts.LocationPoint{{Lat: 10, Lng: 10, Country: "US"}}
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) { return rounds[0], nil })
	_, err := e.CreateMatch("m-preview", []string{"u1", "u2"}, map[string]contracts.PlayerProfile{
		"u1": {UserID: "u1", DisplayName: "One", MMR: 1600, RatingRD: 350},
		"u2": {UserID: "u2", DisplayName: "Two", MMR: 2200, RatingRD: 350},
	})
	if err != nil {
		t.Fatal(err)
	}
	snap, err := e.GetSnapshot("m-preview")
	if err != nil {
		t.Fatal(err)
	}
	if snap.RatingPreview["u1"].Win > 80 || snap.RatingPreview["u1"].Lose < -80 {
		t.Fatalf("expected capped rating preview for u1, got %+v", snap.RatingPreview["u1"])
	}
	if snap.RatingPreview["u2"].Win > 80 || snap.RatingPreview["u2"].Lose < -80 {
		t.Fatalf("expected capped rating preview for u2, got %+v", snap.RatingPreview["u2"])
	}
}

func TestCreateUnrankedMatchOmitsRatingPreview(t *testing.T) {
	rounds := []contracts.LocationPoint{{Lat: 10, Lng: 10, Country: "US"}}
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) { return rounds[0], nil })
	_, err := e.CreateMatchWithOptions("m-private", []string{"u1", "u2"}, map[string]contracts.PlayerProfile{
		"u1": {UserID: "u1", DisplayName: "One", MMR: 1600, RatingRD: 350},
		"u2": {UserID: "u2", DisplayName: "Two", MMR: 2200, RatingRD: 350},
	}, MatchOptions{Unranked: true})
	if err != nil {
		t.Fatal(err)
	}
	snap, err := e.GetSnapshot("m-private")
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Unranked {
		t.Fatalf("expected unranked snapshot")
	}
	if len(snap.RatingPreview) != 0 {
		t.Fatalf("expected no rating preview, got %+v", snap.RatingPreview)
	}
}

func TestTeamDuelUsesBestGuessPerTeamForSharedHP(t *testing.T) {
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) {
		return contracts.LocationPoint{Lat: 10, Lng: 10, Country: "US"}, nil
	})
	m, err := e.CreateMatchWithOptions("m-team", []string{"a1", "a2", "b1"}, nil, MatchOptions{
		Mode: contracts.ModeTeamDuel,
		Teams: map[string]string{
			"a1": "a",
			"a2": "a",
			"b1": "b",
		},
	})
	if err != nil {
		t.Fatalf("create team duel: %v", err)
	}
	m.RoundStartedAt = time.Now().Add(-roundIntro)
	snap, err := e.GetSnapshot("m-team")
	if err != nil {
		t.Fatal(err)
	}
	roundID := snap.CurrentRound.RoundID
	guesses := []contracts.GuessPayload{
		{UserID: "a1", MatchID: "m-team", RoundID: roundID, Lat: 80, Lng: 80, Finalize: true},
		{UserID: "a2", MatchID: "m-team", RoundID: roundID, Lat: 10.01, Lng: 10.01, Finalize: true},
		{UserID: "b1", MatchID: "m-team", RoundID: roundID, Lat: 50, Lng: 50, Finalize: true},
	}
	for _, guess := range guesses {
		snap, err = e.SubmitGuess(guess)
		if err != nil {
			t.Fatal(err)
		}
	}
	if snap.LastRoundResult == nil || snap.LastRoundResult.Teams["a"].RepresentativeUserID != "a2" {
		t.Fatalf("expected a2 to represent team a, got %+v", snap.LastRoundResult)
	}
	if snap.Teams["b"].HP >= startingHP {
		t.Fatalf("expected team b hp to drop, got %+v", snap.Teams["b"])
	}
	if snap.Players["b1"].HP != snap.Teams["b"].HP {
		t.Fatalf("expected player hp to mirror team hp")
	}
}

func TestFreeForAllAccumulatesPointsAndEndsAfterFiveRounds(t *testing.T) {
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) {
		return contracts.LocationPoint{Lat: 10, Lng: 10, Country: "US"}, nil
	})
	m, err := e.CreateMatchWithOptions("m-ffa", []string{"u1", "u2", "u3"}, nil, MatchOptions{Mode: contracts.ModeFreeForAll})
	if err != nil {
		t.Fatalf("create ffa: %v", err)
	}
	var snap *contracts.MatchSnapshot
	for round := 0; round < ffaRounds; round++ {
		m.RoundStartedAt = time.Now().Add(-roundIntro)
		current, err := e.GetSnapshot("m-ffa")
		if err != nil {
			t.Fatal(err)
		}
		for _, userID := range []string{"u1", "u2", "u3"} {
			snap, err = e.SubmitGuess(contracts.GuessPayload{
				UserID: userID, MatchID: "m-ffa", RoundID: current.CurrentRound.RoundID, Lat: 10, Lng: 10, Finalize: true,
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		if round < ffaRounds-1 {
			e.mu.Lock()
			e.advanceRound(m)
			e.mu.Unlock()
		}
	}
	if snap.State != contracts.MatchEnded {
		t.Fatalf("expected ffa to end after %d rounds, got %s", ffaRounds, snap.State)
	}
	if snap.Players["u1"].TotalScore <= 0 || snap.Players["u2"].TotalScore <= 0 {
		t.Fatalf("expected accumulated scores, got %+v", snap.Players)
	}
	if snap.Teams != nil {
		t.Fatalf("ffa should not expose teams")
	}
}

func TestDisconnectResume(t *testing.T) {
	rounds := []contracts.LocationPoint{{Lat: 1, Lng: 1, Country: "US"}}
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) { return rounds[0], nil })
	m, err := e.CreateMatch("m2", []string{"u1", "u2"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.RoundStartedAt = time.Now().Add(-roundIntro)
	snap, err := e.MarkDisconnected("m2", "u1")
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Players["u1"].Disconnected {
		t.Fatalf("player should be disconnected")
	}
	_, err = e.MarkResumed("m2", "u1")
	if err != nil {
		t.Fatal(err)
	}
	snap, _ = e.GetSnapshot("m2")
	if snap.Players["u1"].Disconnected {
		t.Fatalf("player should be resumed")
	}
	if snap.RoundPhase != contracts.RoundPhaseLive {
		t.Fatalf("expected round live phase, got %s", snap.RoundPhase)
	}
}

func TestDisconnectForfeitAfterGrace(t *testing.T) {
	rounds := []contracts.LocationPoint{{Lat: 1, Lng: 1, Country: "US"}}
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) { return rounds[0], nil })
	m, err := e.CreateMatch("m3", []string{"u1", "u2"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.RoundStartedAt = time.Now().Add(-roundIntro)
	if _, err := e.MarkDisconnected("m3", "u1"); err != nil {
		t.Fatal(err)
	}
	m.Players["u1"].DisconnectDue = time.Now().Add(-1 * time.Second).UnixMilli()
	e.Tick()
	snap, _ := e.GetSnapshot("m3")
	if snap.State != contracts.MatchEnded {
		t.Fatalf("expected match ended after disconnect grace")
	}
	if snap.Players["u1"].HP != 0 {
		t.Fatalf("expected forfeited hp=0")
	}
}

func TestUnrankedDisconnectDoesNotEndMatch(t *testing.T) {
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) {
		return contracts.LocationPoint{Lat: 1, Lng: 1, Country: "US"}, nil
	})
	m, err := e.CreateMatchWithOptions("m-unranked-disconnect", []string{"u1", "u2"}, nil, MatchOptions{Unranked: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.MarkDisconnected(m.ID, "u1"); err != nil {
		t.Fatal(err)
	}
	if m.Players["u1"].DisconnectDue != 0 {
		t.Fatalf("unranked disconnect should not have a forfeit deadline")
	}
	e.Tick()
	snap, err := e.GetSnapshot(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if snap.State != contracts.MatchLive {
		t.Fatalf("unranked match ended after disconnect: %s", snap.State)
	}
	if !snap.Players["u1"].Disconnected {
		t.Fatal("disconnected player should remain marked for reconnect")
	}
}

func TestImmediateForfeitEndsMatch(t *testing.T) {
	rounds := []contracts.LocationPoint{{Lat: 1, Lng: 1, Country: "US"}}
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) { return rounds[0], nil })
	_, err := e.CreateMatch("m-forfeit", []string{"u1", "u2"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := e.Forfeit("m-forfeit", "u1")
	if err != nil {
		t.Fatal(err)
	}
	if snap.State != contracts.MatchEnded {
		t.Fatalf("expected match ended after forfeit")
	}
	if snap.Players["u1"].HP != 0 {
		t.Fatalf("expected forfeiting player hp=0")
	}
	if snap.Players["u2"].HP != 6000 {
		t.Fatalf("expected opponent hp to remain full, got %d", snap.Players["u2"].HP)
	}
}

func TestRoundResultPhaseAndDamageFromScoreDelta(t *testing.T) {
	rounds := []contracts.LocationPoint{
		{Lat: 0, Lng: 0, Country: "US"},
		{Lat: 5, Lng: 5, Country: "US"},
	}
	e := New(func(_ string, i int) (contracts.LocationPoint, error) { return rounds[i%len(rounds)], nil })
	m, err := e.CreateMatch("m4", []string{"u1", "u2"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.RoundStartedAt = time.Now().Add(-roundIntro)
	snap, err := e.GetSnapshot("m4")
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.SubmitGuess(contracts.GuessPayload{UserID: "u1", MatchID: "m4", RoundID: snap.CurrentRound.RoundID, Lat: 0, Lng: 0, Finalize: true})
	if err != nil {
		t.Fatal(err)
	}
	resultSnap, err := e.SubmitGuess(contracts.GuessPayload{UserID: "u2", MatchID: "m4", RoundID: snap.CurrentRound.RoundID, Lat: 60, Lng: 60, Finalize: true})
	if err != nil {
		t.Fatal(err)
	}
	if resultSnap.Phase != contracts.PhaseRoundResult {
		t.Fatalf("expected round result phase, got %s", resultSnap.Phase)
	}
	if resultSnap.LastRoundResult == nil {
		t.Fatalf("expected last round result")
	}
	if len(resultSnap.RoundResults) != 1 {
		t.Fatalf("expected round history length 1, got %d", len(resultSnap.RoundResults))
	}
	r1 := resultSnap.LastRoundResult.Players["u1"]
	r2 := resultSnap.LastRoundResult.Players["u2"]
	if r1.Score <= r2.Score {
		t.Fatalf("expected u1 to outscore u2")
	}
	wantDamage := r1.Score - r2.Score
	if r1.DamageDealt != wantDamage || r2.DamageTaken != wantDamage {
		t.Fatalf("expected damage=%d got dealt=%d taken=%d", wantDamage, r1.DamageDealt, r2.DamageTaken)
	}
	if resultSnap.Players["u2"].HP != startingHP-wantDamage {
		t.Fatalf("expected hp=%d got %d", startingHP-wantDamage, resultSnap.Players["u2"].HP)
	}
}

func TestRoundHistoryPersistsAcrossAdvance(t *testing.T) {
	rounds := []contracts.LocationPoint{
		{Lat: 0, Lng: 0, Country: "US"},
		{Lat: 10, Lng: 10, Country: "US"},
	}
	e := New(func(_ string, i int) (contracts.LocationPoint, error) { return rounds[i%len(rounds)], nil })
	m, err := e.CreateMatch("m-history", []string{"u1", "u2"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.RoundStartedAt = time.Now().Add(-roundIntro)
	snap, err := e.GetSnapshot("m-history")
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.SubmitGuess(contracts.GuessPayload{UserID: "u1", MatchID: "m-history", RoundID: snap.CurrentRound.RoundID, Lat: 0, Lng: 0, Finalize: true})
	if err != nil {
		t.Fatal(err)
	}
	resultSnap, err := e.SubmitGuess(contracts.GuessPayload{UserID: "u2", MatchID: "m-history", RoundID: snap.CurrentRound.RoundID, Lat: 20, Lng: 20, Finalize: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(resultSnap.RoundResults) != 1 {
		t.Fatalf("expected one round result before advance, got %d", len(resultSnap.RoundResults))
	}
	m.IntermissionUntil = time.Now().Add(-time.Millisecond)
	e.Tick()
	nextSnap, err := e.GetSnapshot("m-history")
	if err != nil {
		t.Fatal(err)
	}
	if nextSnap.CurrentRound == nil || nextSnap.CurrentRound.RoundNumber != 2 {
		t.Fatalf("expected round 2 after advance, got %+v", nextSnap.CurrentRound)
	}
	if len(nextSnap.RoundResults) != 1 {
		t.Fatalf("expected prior round history to persist, got %d", len(nextSnap.RoundResults))
	}
}

func liveRoundID(t *testing.T, e *Engine, m *Match) string {
	t.Helper()
	m.RoundStartedAt = time.Now().Add(-roundIntro)
	snap, err := e.GetSnapshot(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if snap.CurrentRound == nil {
		t.Fatalf("expected a live round, got %+v", snap)
	}
	return snap.CurrentRound.RoundID
}

func advanceToNextRound(t *testing.T, e *Engine, m *Match) {
	t.Helper()
	m.IntermissionUntil = time.Now().Add(-time.Millisecond)
	e.Tick()
}

func playRound(t *testing.T, e *Engine, m *Match, guesses map[string][2]float64) *contracts.MatchSnapshot {
	t.Helper()
	roundID := liveRoundID(t, e, m)
	var snap *contracts.MatchSnapshot
	var err error
	for userID, latLng := range guesses {
		snap, err = e.SubmitGuess(contracts.GuessPayload{
			UserID:   userID,
			MatchID:  m.ID,
			RoundID:  roundID,
			Lat:      latLng[0],
			Lng:      latLng[1],
			Finalize: true,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if snap == nil || snap.LastRoundResult == nil {
		t.Fatalf("expected a resolved round result, got %+v", snap)
	}
	return snap
}

func TestDuelDamageMultiplierAccumulatesPerWin(t *testing.T) {
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) {
		return contracts.LocationPoint{Lat: 0, Lng: 0, Country: "US"}, nil
	})
	m, err := e.CreateMatch("m-mult-duel", []string{"u1", "u2"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	round1 := playRound(t, e, m, map[string][2]float64{
		"u1": {0, 0},
		"u2": {0.5, 0.5},
	})
	winner1 := round1.LastRoundResult.Players["u1"]
	loser1 := round1.LastRoundResult.Players["u2"]
	if winner1.Score <= loser1.Score {
		t.Fatalf("expected u1 to outscore u2, got %d vs %d", winner1.Score, loser1.Score)
	}
	scoreDelta1 := winner1.Score - loser1.Score
	if winner1.DamageMultiplier != 1.0 || loser1.DamageMultiplier != 1.0 {
		t.Fatalf("expected 1x applied in round 1, got %v and %v", winner1.DamageMultiplier, loser1.DamageMultiplier)
	}
	if winner1.DamageDealt != scoreDelta1 {
		t.Fatalf("expected round 1 damage=%d got %d", scoreDelta1, winner1.DamageDealt)
	}
	if round1.Players["u1"].DamageMultiplier != 1.0 {
		t.Fatalf("expected the winner multiplier to still read 1x while the result is shown, got %v", round1.Players["u1"].DamageMultiplier)
	}

	advanceToNextRound(t, e, m)
	liveSnap, err := e.GetSnapshot("m-mult-duel")
	if err != nil {
		t.Fatal(err)
	}
	if liveSnap.Players["u1"].DamageMultiplier != 1.5 {
		t.Fatalf("expected winner multiplier 1.5 in round 2, got %v", liveSnap.Players["u1"].DamageMultiplier)
	}
	if liveSnap.Players["u2"].DamageMultiplier != 1.0 {
		t.Fatalf("expected loser multiplier to stay 1, got %v", liveSnap.Players["u2"].DamageMultiplier)
	}

	hpBeforeRound2 := liveSnap.Players["u2"].HP
	round2 := playRound(t, e, m, map[string][2]float64{
		"u1": {0, 0},
		"u2": {0.5, 0.5},
	})
	winner2 := round2.LastRoundResult.Players["u1"]
	loser2 := round2.LastRoundResult.Players["u2"]
	scoreDelta2 := winner2.Score - loser2.Score
	wantDamage2 := scaledDamage(scoreDelta2, 1.5)
	if winner2.DamageMultiplier != 1.5 {
		t.Fatalf("expected applied multiplier 1.5 in round 2, got %v", winner2.DamageMultiplier)
	}
	if winner2.DamageDealt != wantDamage2 || loser2.DamageTaken != wantDamage2 {
		t.Fatalf("expected round 2 damage=%d got dealt=%d taken=%d", wantDamage2, winner2.DamageDealt, loser2.DamageTaken)
	}
	if round2.Players["u2"].HP != hpBeforeRound2-wantDamage2 {
		t.Fatalf("expected hp=%d got %d", hpBeforeRound2-wantDamage2, round2.Players["u2"].HP)
	}

	advanceToNextRound(t, e, m)
	round3Snap, err := e.GetSnapshot("m-mult-duel")
	if err != nil {
		t.Fatal(err)
	}
	if round3Snap.Players["u1"].DamageMultiplier != 2.0 {
		t.Fatalf("expected winner multiplier 2 after a second win, got %v", round3Snap.Players["u1"].DamageMultiplier)
	}
	if round3Snap.Players["u2"].DamageMultiplier != 1.0 {
		t.Fatalf("expected loser multiplier to stay 1, got %v", round3Snap.Players["u2"].DamageMultiplier)
	}
}

func TestDuelTieRaisesBothDamageMultipliers(t *testing.T) {
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) {
		return contracts.LocationPoint{Lat: 0, Lng: 0, Country: "US"}, nil
	})
	m, err := e.CreateMatch("m-mult-tie", []string{"u1", "u2"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	tied := playRound(t, e, m, map[string][2]float64{
		"u1": {0.5, 0.5},
		"u2": {0.5, 0.5},
	})
	if tied.LastRoundResult.Players["u1"].DamageDealt != 0 || tied.LastRoundResult.Players["u2"].DamageDealt != 0 {
		t.Fatalf("expected no damage on a tie, got %+v", tied.LastRoundResult.Players)
	}

	advanceToNextRound(t, e, m)
	snap, err := e.GetSnapshot("m-mult-tie")
	if err != nil {
		t.Fatal(err)
	}
	if snap.Players["u1"].DamageMultiplier != 1.5 || snap.Players["u2"].DamageMultiplier != 1.5 {
		t.Fatalf("expected both multipliers at 1.5 after a tie, got %v and %v",
			snap.Players["u1"].DamageMultiplier, snap.Players["u2"].DamageMultiplier)
	}
}

func TestTeamDuelDamageMultiplierAccumulatesPerWin(t *testing.T) {
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) {
		return contracts.LocationPoint{Lat: 0, Lng: 0, Country: "US"}, nil
	})
	m, err := e.CreateMatchWithOptions("m-mult-team", []string{"a1", "a2", "b1"}, nil, MatchOptions{
		Mode: contracts.ModeTeamDuel,
		Teams: map[string]string{
			"a1": "a",
			"a2": "a",
			"b1": "b",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	round1 := playRound(t, e, m, map[string][2]float64{
		"a1": {0.5, 0.5},
		"a2": {0, 0},
		"b1": {0.5, 0.5},
	})
	teamA1 := round1.LastRoundResult.Teams["a"]
	teamB1 := round1.LastRoundResult.Teams["b"]
	if teamA1.Score <= teamB1.Score {
		t.Fatalf("expected team a to outscore team b, got %d vs %d", teamA1.Score, teamB1.Score)
	}
	if teamA1.DamageMultiplier != 1.0 {
		t.Fatalf("expected 1x applied in round 1, got %v", teamA1.DamageMultiplier)
	}
	if teamA1.DamageDealt != teamA1.Score-teamB1.Score {
		t.Fatalf("expected round 1 damage=%d got %d", teamA1.Score-teamB1.Score, teamA1.DamageDealt)
	}
	if round1.Teams["a"].DamageMultiplier != 1.0 {
		t.Fatalf("expected team multiplier to still read 1x while the result is shown, got %v", round1.Teams["a"].DamageMultiplier)
	}

	advanceToNextRound(t, e, m)
	liveSnap, err := e.GetSnapshot("m-mult-team")
	if err != nil {
		t.Fatal(err)
	}
	if liveSnap.Teams["a"].DamageMultiplier != 1.5 {
		t.Fatalf("expected winning team multiplier 1.5, got %v", liveSnap.Teams["a"].DamageMultiplier)
	}
	if liveSnap.Teams["b"].DamageMultiplier != 1.0 {
		t.Fatalf("expected losing team multiplier to stay 1, got %v", liveSnap.Teams["b"].DamageMultiplier)
	}
	for _, userID := range []string{"a1", "a2"} {
		if liveSnap.Players[userID].DamageMultiplier != 1.5 {
			t.Fatalf("expected %s to mirror the team multiplier, got %v", userID, liveSnap.Players[userID].DamageMultiplier)
		}
	}
	if liveSnap.Players["b1"].DamageMultiplier != 1.0 {
		t.Fatalf("expected b1 to mirror the losing team multiplier, got %v", liveSnap.Players["b1"].DamageMultiplier)
	}

	hpBeforeRound2 := liveSnap.Teams["b"].HP
	round2 := playRound(t, e, m, map[string][2]float64{
		"a1": {0.5, 0.5},
		"a2": {0, 0},
		"b1": {0.5, 0.5},
	})
	teamA2 := round2.LastRoundResult.Teams["a"]
	teamB2 := round2.LastRoundResult.Teams["b"]
	wantDamage := scaledDamage(teamA2.Score-teamB2.Score, 1.5)
	if teamA2.DamageMultiplier != 1.5 {
		t.Fatalf("expected applied team multiplier 1.5 in round 2, got %v", teamA2.DamageMultiplier)
	}
	if teamA2.DamageDealt != wantDamage || teamB2.DamageTaken != wantDamage {
		t.Fatalf("expected round 2 damage=%d got dealt=%d taken=%d", wantDamage, teamA2.DamageDealt, teamB2.DamageTaken)
	}
	if round2.Teams["b"].HP != hpBeforeRound2-wantDamage {
		t.Fatalf("expected team b hp=%d got %d", hpBeforeRound2-wantDamage, round2.Teams["b"].HP)
	}
}

func TestBothDisconnectedEndsMatchAsStale(t *testing.T) {
	rounds := []contracts.LocationPoint{{Lat: 1, Lng: 1, Country: "US"}}
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) { return rounds[0], nil })
	m, err := e.CreateMatch("m5", []string{"u1", "u2"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.RoundStartedAt = time.Now().Add(-roundIntro)
	if _, err := e.MarkDisconnected("m5", "u1"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.MarkDisconnected("m5", "u2"); err != nil {
		t.Fatal(err)
	}
	m.Players["u1"].DisconnectDue = time.Now().Add(-2 * time.Second).UnixMilli()
	m.Players["u2"].DisconnectDue = time.Now().Add(-2 * time.Second).UnixMilli()
	e.Tick()
	snap, _ := e.GetSnapshot("m5")
	if snap.State != contracts.MatchEnded {
		t.Fatalf("expected match ended")
	}
	if snap.Players["u1"].HP != 0 || snap.Players["u2"].HP != 0 {
		t.Fatalf("expected both players hp=0 on stale dual disconnect")
	}
}

func TestRoundIntroBlocksEarlyGuesses(t *testing.T) {
	rounds := []contracts.LocationPoint{{Lat: 1, Lng: 1, Country: "US"}}
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) { return rounds[0], nil })
	_, err := e.CreateMatch("m6", []string{"u1", "u2"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := e.GetSnapshot("m6")
	if err != nil {
		t.Fatal(err)
	}
	if snap.RoundPhase != contracts.RoundPhaseIntro {
		t.Fatalf("expected round intro phase, got %s", snap.RoundPhase)
	}
	beforeSeq := snap.EventSequence
	next, err := e.SubmitGuess(contracts.GuessPayload{UserID: "u1", MatchID: "m6", RoundID: snap.CurrentRound.RoundID, Lat: 1, Lng: 1, Finalize: true})
	if err != nil {
		t.Fatal(err)
	}
	if next.EventSequence != beforeSeq {
		t.Fatalf("expected early guess to be ignored during intro")
	}
}

func TestTickReportsIntroToLiveTransitionOnce(t *testing.T) {
	rounds := []contracts.LocationPoint{{Lat: 1, Lng: 1, Country: "US"}}
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) { return rounds[0], nil })
	m, err := e.CreateMatch("m-intro-live", []string{"u1", "u2"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.RoundStartedAt = time.Now().Add(-(roundIntro + time.Millisecond))
	before := m.EventSeq

	changed := e.Tick()
	if len(changed) != 1 || changed[0] != "m-intro-live" {
		t.Fatalf("expected intro-to-live change, got %+v", changed)
	}
	snap, err := e.GetSnapshot("m-intro-live")
	if err != nil {
		t.Fatal(err)
	}
	if snap.EventSequence != before+1 {
		t.Fatalf("expected event sequence to advance once, got %d from %d", snap.EventSequence, before)
	}
	if snap.RoundPhase != contracts.RoundPhaseLive {
		t.Fatalf("expected round live phase, got %s", snap.RoundPhase)
	}
	if changed := e.Tick(); len(changed) != 0 {
		t.Fatalf("expected intro transition to be reported once, got %+v", changed)
	}
}

func TestFinalizedGuessCannotBeOverwritten(t *testing.T) {
	rounds := []contracts.LocationPoint{{Lat: 1, Lng: 1, Country: "US"}}
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) { return rounds[0], nil })
	m, err := e.CreateMatch("m8", []string{"u1", "u2"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.RoundStartedAt = time.Now().Add(-roundIntro)
	snap, err := e.GetSnapshot("m8")
	if err != nil {
		t.Fatal(err)
	}
	finalized, err := e.SubmitGuess(contracts.GuessPayload{
		UserID: "u1", MatchID: "m8", RoundID: snap.CurrentRound.RoundID, Lat: 10, Lng: 10, Finalize: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	beforeSeq := finalized.EventSequence
	next, err := e.SubmitGuess(contracts.GuessPayload{
		UserID: "u1", MatchID: "m8", RoundID: snap.CurrentRound.RoundID, Lat: 20, Lng: 20, Finalize: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if next.EventSequence != beforeSeq {
		t.Fatalf("expected post-finalize guess mutation to be ignored")
	}
	if !next.Players["u1"].Finalized {
		t.Fatalf("expected player to remain finalized")
	}
	if next.Players["u1"].LastGuessLat != 10 || next.Players["u1"].LastGuessLng != 10 {
		t.Fatalf("expected finalized guess coordinates to remain unchanged")
	}
}

func TestEarlyFinalizeCapsOpponentTimer(t *testing.T) {
	rounds := []contracts.LocationPoint{{Lat: 1, Lng: 1, Country: "US"}}
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) { return rounds[0], nil })
	m, err := e.CreateMatch("m9", []string{"u1", "u2"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.RoundStartedAt = time.Now().Add(-roundIntro)
	if !m.RoundDeadline.IsZero() {
		t.Fatalf("expected round timer to be idle before first finalized guess")
	}
	snap, err := e.GetSnapshot("m9")
	if err != nil {
		t.Fatal(err)
	}
	if snap.CurrentRound == nil || snap.CurrentRound.TimerStarted {
		t.Fatalf("expected snapshot timer to be idle before first finalized guess")
	}
	if snap.RoundMSLeft != 0 || snap.PhaseEndsAt != 0 {
		t.Fatalf("expected untimed live round, got msLeft=%d phaseEndsAt=%d", snap.RoundMSLeft, snap.PhaseEndsAt)
	}
	start := time.Now()
	next, err := e.SubmitGuess(contracts.GuessPayload{
		UserID: "u1", MatchID: "m9", RoundID: snap.CurrentRound.RoundID, Lat: 1, Lng: 1, Finalize: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if next.CurrentRound == nil || !next.CurrentRound.TimerStarted {
		t.Fatalf("expected timer to start after finalized guess")
	}
	remaining := time.Until(next.CurrentRound.RoundDeadline)
	pressureDuration := time.Duration(contracts.DefaultPressureTimeMS) * time.Millisecond
	if remaining > pressureDuration+500*time.Millisecond {
		t.Fatalf("expected timer capped near %s, got %s remaining", pressureDuration, remaining)
	}
	if remaining < pressureDuration-1*time.Second {
		t.Fatalf("expected timer to remain close to %s, got %s remaining", pressureDuration, remaining)
	}
	if next.CurrentRound.RoundDeadline.Before(start.Add(pressureDuration - 1500*time.Millisecond)) {
		t.Fatalf("expected deadline to be pressure window after finalize")
	}
}

func TestPlaceGuessDoesNotStartRoundTimer(t *testing.T) {
	rounds := []contracts.LocationPoint{{Lat: 1, Lng: 1, Country: "US"}}
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) { return rounds[0], nil })
	m, err := e.CreateMatch("m-place", []string{"u1", "u2"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.RoundStartedAt = time.Now().Add(-roundIntro)
	snap, err := e.GetSnapshot("m-place")
	if err != nil {
		t.Fatal(err)
	}
	next, err := e.SubmitGuess(contracts.GuessPayload{
		UserID: "u1", MatchID: "m-place", RoundID: snap.CurrentRound.RoundID, Lat: 1, Lng: 1, Finalize: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if next.CurrentRound == nil || next.CurrentRound.TimerStarted {
		t.Fatalf("expected placed guess to leave round timer idle")
	}
	if next.RoundMSLeft != 0 || next.PhaseEndsAt != 0 {
		t.Fatalf("expected placed guess to leave live round untimed, got msLeft=%d phaseEndsAt=%d", next.RoundMSLeft, next.PhaseEndsAt)
	}
}

func TestActiveUntimedRoundDoesNotResolveAtIdleCap(t *testing.T) {
	rounds := []contracts.LocationPoint{{Lat: 1, Lng: 1, Country: "US"}}
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) { return rounds[0], nil })
	m, err := e.CreateMatch("m-active", []string{"u1", "u2"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.RoundStartedAt = time.Now().Add(-(roundIntro + roundIdleCap + time.Minute))
	m.LastActivity = time.Now()
	e.Tick()
	snap, err := e.GetSnapshot("m-active")
	if err != nil {
		t.Fatal(err)
	}
	if snap.Phase != contracts.PhaseLive {
		t.Fatalf("expected active untimed round to stay live, got phase=%s", snap.Phase)
	}
	if snap.LastRoundResult != nil {
		t.Fatalf("did not expect a round result for an active untimed round")
	}
}

func TestIdleRoundCapResolvesUntimedRound(t *testing.T) {
	rounds := []contracts.LocationPoint{{Lat: 1, Lng: 1, Country: "US"}}
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) { return rounds[0], nil })
	m, err := e.CreateMatch("m-idle", []string{"u1", "u2"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	idlePast := time.Now().Add(-(roundIntro + roundIdleCap + time.Second))
	m.RoundStartedAt = idlePast
	m.LastActivity = idlePast
	e.Tick()
	snap, err := e.GetSnapshot("m-idle")
	if err != nil {
		t.Fatal(err)
	}
	if snap.Phase != contracts.PhaseRoundResult {
		t.Fatalf("expected idle untimed round to resolve, got phase=%s", snap.Phase)
	}
	if snap.LastRoundResult == nil {
		t.Fatalf("expected round result after idle cap")
	}
}

func TestLateFinalizeDoesNotExtendShortTimer(t *testing.T) {
	rounds := []contracts.LocationPoint{{Lat: 1, Lng: 1, Country: "US"}}
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) { return rounds[0], nil })
	m, err := e.CreateMatch("m10", []string{"u1", "u2"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.RoundStartedAt = time.Now().Add(-roundIntro)
	expectedDeadline := time.Now().Add(5 * time.Second)
	m.RoundDeadline = expectedDeadline
	snap, err := e.GetSnapshot("m10")
	if err != nil {
		t.Fatal(err)
	}
	next, err := e.SubmitGuess(contracts.GuessPayload{
		UserID: "u1", MatchID: "m10", RoundID: snap.CurrentRound.RoundID, Lat: 1, Lng: 1, Finalize: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	diff := next.CurrentRound.RoundDeadline.Sub(expectedDeadline)
	if diff < -200*time.Millisecond || diff > 200*time.Millisecond {
		t.Fatalf("expected short deadline unchanged, got drift %s", diff)
	}
}

func TestMatchEndsAtMaxRoundsCap(t *testing.T) {
	rounds := []contracts.LocationPoint{{Lat: 1, Lng: 1, Country: "US"}}
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) { return rounds[0], nil })
	m, err := e.CreateMatch("m7", []string{"u1", "u2"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.CurrentIndex = maxRounds - 1
	m.RoundID = roundID("m7", maxRounds)
	m.RoundStartedAt = time.Now().Add(-roundIntro)
	m.RoundDeadline = time.Now().Add(roundDuration)

	snap, err := e.GetSnapshot("m7")
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.SubmitGuess(contracts.GuessPayload{
		UserID: "u1", MatchID: "m7", RoundID: snap.CurrentRound.RoundID, Lat: 1, Lng: 1, Finalize: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	finalSnap, err := e.SubmitGuess(contracts.GuessPayload{
		UserID: "u2", MatchID: "m7", RoundID: snap.CurrentRound.RoundID, Lat: 1, Lng: 1, Finalize: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if finalSnap.State != contracts.MatchEnded {
		t.Fatalf("expected match ended at max rounds cap")
	}
}

func TestRoundScoreGeoGuessrCurve(t *testing.T) {
	if got := roundScore(0); got != maxScore {
		t.Fatalf("distance 0km score=%d want=%d", got, maxScore)
	}
	if got := roundScore(perfectGuessKm); got != maxScore {
		t.Fatalf("distance %.3fkm score=%d want=%d", perfectGuessKm, got, maxScore)
	}
	for _, tc := range []struct {
		distanceKm float64
		want       int
	}{
		{distanceKm: 4.8, want: 4984},
		{distanceKm: 19, want: 4936},
		{distanceKm: 65, want: 4787},
		{distanceKm: 426, want: 3759},
	} {
		if got := roundScore(tc.distanceKm); got != tc.want {
			t.Fatalf("distance %.1fkm score=%d want=%d", tc.distanceKm, got, tc.want)
		}
	}
	if got := roundScore(maxDistanceKm); got != 0 {
		t.Fatalf("distance max score=%d want=0", got)
	}
}

func TestRoundScoreIsMonotonicDecreasing(t *testing.T) {
	distances := []float64{0, 0.03, 1, 30, 100, 1000, 5000, 10000, maxDistanceKm}
	prev := maxScore + 1
	for _, d := range distances {
		got := roundScore(d)
		if got > prev {
			t.Fatalf("score increased at distance %.2fkm: got=%d prev=%d", d, got, prev)
		}
		prev = got
	}
}

func TestLiveTickAppliesMultiplierBonus(t *testing.T) {
	e := New(func(_ string, _ int) (contracts.LocationPoint, error) {
		return contracts.LocationPoint{Lat: 0, Lng: 0, Country: "US"}, nil
	})
	m, err := e.CreateMatch("m-live-tick", []string{"u1", "u2"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	m.RoundStartedAt = time.Now().Add(-roundIntro)
	rid := liveRoundID(t, e, m)
	for _, g := range []contracts.GuessPayload{
		{UserID: "u1", MatchID: "m-live-tick", RoundID: rid, Lat: 0, Lng: 0, Finalize: true},
		{UserID: "u2", MatchID: "m-live-tick", RoundID: rid, Lat: 0.5, Lng: 0.5, Finalize: true},
	} {
		if _, err := e.SubmitGuess(g); err != nil {
			t.Fatal(err)
		}
	}
	if !m.PendingAdvance {
		t.Fatalf("expected match to be pending advance after round 1")
	}

	m.IntermissionUntil = time.Now().Add(-time.Millisecond)
	e.Tick()

	snap, err := e.GetSnapshot("m-live-tick")
	if err != nil {
		t.Fatal(err)
	}
	if snap.Players["u1"].DamageMultiplier != 1.5 {
		t.Fatalf("expected live Tick to apply the win bonus (1.5), got %v",
			snap.Players["u1"].DamageMultiplier)
	}
	if snap.Players["u2"].DamageMultiplier != 1.0 {
		t.Fatalf("expected loser to stay at 1.0, got %v", snap.Players["u2"].DamageMultiplier)
	}
}
