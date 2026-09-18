package sessionpolicy

import (
	"testing"

	"geoduels/pkg/contracts"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name         string
		existing     contracts.MatchMode
		requested    contracts.MatchMode
		wantDecision Decision
	}{
		{name: "singleplayer replaces singleplayer", existing: contracts.ModeSingleplayer, requested: contracts.ModeSingleplayer, wantDecision: DecisionReplace},
		{name: "duel rejects singleplayer", existing: contracts.ModeDuel, requested: contracts.ModeSingleplayer, wantDecision: DecisionReject},
		{name: "duel resumes duel", existing: contracts.ModeDuel, requested: contracts.ModeDuel, wantDecision: DecisionResume},
		{name: "singleplayer yields to duel", existing: contracts.ModeSingleplayer, requested: contracts.ModeDuel, wantDecision: DecisionReplace},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Resolve(tt.existing, tt.requested); got != tt.wantDecision {
				t.Fatalf("expected %q, got %q", tt.wantDecision, got)
			}
		})
	}
}
