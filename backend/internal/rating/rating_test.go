package rating

import (
	"math"
	"testing"
	"time"
)

func TestCalculateDuelUpdatesMovesUncertainPlayersMoreThanEstablishedPlayers(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	uncertain1, uncertain2 := CalculateDuelUpdates(
		State{MMR: 1500, RD: InitialRatingRD, UpdatedAt: now},
		State{MMR: 1500, RD: InitialRatingRD, UpdatedAt: now},
		"p1",
		now,
	)
	established1, established2 := CalculateDuelUpdates(
		State{MMR: 1500, RD: MinimumRatingRD, UpdatedAt: now},
		State{MMR: 1500, RD: MinimumRatingRD, UpdatedAt: now},
		"p1",
		now,
	)

	if uncertain1.Delta <= established1.Delta {
		t.Fatalf("expected high-RD winner to gain more than established winner, got %d and %d", uncertain1.Delta, established1.Delta)
	}
	if math.Abs(float64(uncertain2.Delta)) <= math.Abs(float64(established2.Delta)) {
		t.Fatalf("expected high-RD loser to lose more than established loser, got %d and %d", uncertain2.Delta, established2.Delta)
	}
	if uncertain1.RD >= InitialRatingRD || uncertain2.RD >= InitialRatingRD {
		t.Fatalf("expected high-RD players to become more certain, got %.2f and %.2f", uncertain1.RD, uncertain2.RD)
	}
}

func TestCalculateDuelUpdatesFavoriteWinStillMovesEstablishedPlayers(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	p1, p2 := CalculateDuelUpdates(
		State{MMR: 1579, RD: MinimumRatingRD, UpdatedAt: now},
		State{MMR: 1363, RD: MinimumRatingRD, UpdatedAt: now},
		"p1",
		now,
	)

	if p1.Delta < 10 || p1.Delta > 20 {
		t.Fatalf("expected established favorite to gain a moderate amount, got %d", p1.Delta)
	}
	if p2.Delta > -10 || p2.Delta < -20 {
		t.Fatalf("expected established underdog to lose a moderate amount, got %d", p2.Delta)
	}
}

func TestCalculateDuelUpdatesEqualEstablishedWinGainsAboutThirty(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	p1, p2 := CalculateDuelUpdates(
		State{MMR: 1500, RD: MinimumRatingRD, UpdatedAt: now},
		State{MMR: 1500, RD: MinimumRatingRD, UpdatedAt: now},
		"p1",
		now,
	)

	if p1.Delta < 28 || p1.Delta > 32 {
		t.Fatalf("expected equal established winner to gain about 30, got %d", p1.Delta)
	}
	if p2.Delta > -28 || p2.Delta < -32 {
		t.Fatalf("expected equal established loser to lose about 30, got %d", p2.Delta)
	}
}

func TestCalculateDuelUpdatesForgivesLowMMRLosses(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	_, loser600 := CalculateDuelUpdates(
		State{MMR: 600, RD: MinimumRatingRD, UpdatedAt: now},
		State{MMR: 600, RD: MinimumRatingRD, UpdatedAt: now},
		"p1",
		now,
	)
	_, loser900 := CalculateDuelUpdates(
		State{MMR: 900, RD: MinimumRatingRD, UpdatedAt: now},
		State{MMR: 900, RD: MinimumRatingRD, UpdatedAt: now},
		"p1",
		now,
	)
	_, loser1000 := CalculateDuelUpdates(
		State{MMR: 1000, RD: MinimumRatingRD, UpdatedAt: now},
		State{MMR: 1000, RD: MinimumRatingRD, UpdatedAt: now},
		"p1",
		now,
	)

	if loser600.Delta > -4 || loser600.Delta < -8 {
		t.Fatalf("expected 600 MMR loser to lose about 20%% of a normal loss, got %d", loser600.Delta)
	}
	if loser900.Delta > -22 || loser900.Delta < -26 {
		t.Fatalf("expected 900 MMR loser to lose about 80%% of a normal loss, got %d", loser900.Delta)
	}
	if loser1000.Delta > -28 || loser1000.Delta < -32 {
		t.Fatalf("expected 1000 MMR loser to take the regular loss, got %d", loser1000.Delta)
	}
}

func TestCalculateDuelUpdatesKeepsMinimumMMRAtFiveHundred(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	_, loser := CalculateDuelUpdates(
		State{MMR: InitialMMR, RD: MinimumRatingRD, UpdatedAt: now},
		State{MMR: InitialMMR, RD: MinimumRatingRD, UpdatedAt: now},
		"p1",
		now,
	)

	if InitialMMR != 500 {
		t.Fatalf("expected initial MMR to be 500, got %d", InitialMMR)
	}
	if loser.MMR != MinimumRankedMMR {
		t.Fatalf("expected loser to stay clamped at %d, got %d", MinimumRankedMMR, loser.MMR)
	}
}

func TestCalculateDuelUpdatesProtectsEstablishedFavoriteAgainstNewUnderdog(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	favorite, underdog := CalculateDuelUpdates(
		State{MMR: 1000, RD: MinimumRatingRD, UpdatedAt: now},
		State{MMR: 500, RD: InitialRatingRD, UpdatedAt: now},
		"p2",
		now,
	)

	if favorite.Delta > -8 || favorite.Delta < -15 {
		t.Fatalf("expected protected favorite loss around 20%% of normal, got %d", favorite.Delta)
	}
	if underdog.Delta != MaxDuelMMRDelta {
		t.Fatalf("expected new underdog upset win to remain capped at +%d, got %d", MaxDuelMMRDelta, underdog.Delta)
	}
}

func TestCalculateDuelUpdatesRestoresUpsetPenaltyAgainstEstablishedUnderdog(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	favorite, _ := CalculateDuelUpdates(
		State{MMR: 1000, RD: MinimumRatingRD, UpdatedAt: now},
		State{MMR: 500, RD: MinimumRatingRD, UpdatedAt: now},
		"p2",
		now,
	)

	if favorite.Delta != -MaxDuelMMRDelta {
		t.Fatalf("expected established underdog upset penalty to cap at -%d, got %d", MaxDuelMMRDelta, favorite.Delta)
	}
}

func TestCalculateDuelUpdatesHighRDExpectedLossMovesMoreThanSettledLoss(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	_, uncertainLoser := CalculateDuelUpdates(
		State{MMR: 1579, RD: MinimumRatingRD, UpdatedAt: now},
		State{MMR: 1363, RD: InitialRatingRD, UpdatedAt: now},
		"p1",
		now,
	)
	_, settledLoser := CalculateDuelUpdates(
		State{MMR: 1579, RD: MinimumRatingRD, UpdatedAt: now},
		State{MMR: 1363, RD: MinimumRatingRD, UpdatedAt: now},
		"p1",
		now,
	)

	if math.Abs(float64(uncertainLoser.Delta)) <= math.Abs(float64(settledLoser.Delta)) {
		t.Fatalf("expected uncertain underdog to move more than settled underdog, got %d and %d", uncertainLoser.Delta, settledLoser.Delta)
	}
	if uncertainLoser.RD >= InitialRatingRD {
		t.Fatalf("expected uncertain underdog RD to decrease after a game, got %.2f", uncertainLoser.RD)
	}
}

func TestCalculateDuelUpdatesCapsUpsetDelta(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	underdog, favorite := CalculateDuelUpdates(
		State{MMR: 1600, RD: InitialRatingRD, UpdatedAt: now},
		State{MMR: 2200, RD: MinimumRatingRD, UpdatedAt: now},
		"p1",
		now,
	)

	if underdog.Delta != MaxDuelMMRDelta {
		t.Fatalf("expected underdog gain to cap at %d, got %d", MaxDuelMMRDelta, underdog.Delta)
	}
	if favorite.Delta < -MaxDuelMMRDelta {
		t.Fatalf("expected favorite loss not to exceed cap %d, got %d", MaxDuelMMRDelta, favorite.Delta)
	}
}

func TestCalculateDuelUpdatesDrawRewardsLowerRatedPlayer(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	p1, p2 := CalculateDuelUpdates(
		State{MMR: 1400, RD: MinimumRatingRD, UpdatedAt: now},
		State{MMR: 1600, RD: MinimumRatingRD, UpdatedAt: now},
		"",
		now,
	)

	if p1.Delta <= 0 {
		t.Fatalf("expected lower rated player to gain on draw, got %d", p1.Delta)
	}
	if p2.Delta >= 0 {
		t.Fatalf("expected higher rated player to lose on draw, got %d", p2.Delta)
	}
}

func TestInflateRatingRD(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	if got := InflateRD(MinimumRatingRD, now.AddDate(0, 0, -1), now); got <= MinimumRatingRD {
		t.Fatalf("expected inactivity to increase RD, got %.2f", got)
	}
	if got := InflateRD(MinimumRatingRD, now.AddDate(-1, 0, 0), now); got != MaximumRatingRD {
		t.Fatalf("expected one inactive year from minimum RD to cap at %.2f, got %.2f", MaximumRatingRD, got)
	}
	if got := InflateRD(MaximumRatingRD, now.AddDate(-1, 0, 0), now); got != MaximumRatingRD {
		t.Fatalf("expected RD to stay capped at %.2f, got %.2f", MaximumRatingRD, got)
	}
}

func TestRatingRDClampsAfterRepeatedGames(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	p1 := State{MMR: 1500, RD: MinimumRatingRD, UpdatedAt: now}
	p2 := State{MMR: 1500, RD: MinimumRatingRD, UpdatedAt: now}

	for i := 0; i < 100; i++ {
		next1, next2 := CalculateDuelUpdates(p1, p2, "p1", now)
		p1 = State{MMR: next1.MMR, RD: next1.RD, UpdatedAt: now}
		p2 = State{MMR: next2.MMR, RD: next2.RD, UpdatedAt: now}
	}

	if p1.RD < MinimumRatingRD || p2.RD < MinimumRatingRD {
		t.Fatalf("expected RD not to fall below %.2f, got %.2f and %.2f", MinimumRatingRD, p1.RD, p2.RD)
	}
}

func TestClampRankedMMR(t *testing.T) {
	if got := ClampRankedMMR(-50); got != MinimumRankedMMR {
		t.Fatalf("expected clamp to %d, got %d", MinimumRankedMMR, got)
	}
}
