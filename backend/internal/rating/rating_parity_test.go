package rating_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"geoduels/internal/rating"
)

// The parity suite runs the production Go rating implementation over a shared,
// committed corpus of inputs and compares against a golden file of outputs.
// The web (TypeScript) implementation is compared against the same golden file
// in web/lib/elo-parity.test.ts, so the two production implementations can
// never silently diverge. Neither test reimplements the rating math.

const sharedDir = "../../../tests/shared"

type parityCase struct {
	ID           string   `json:"id"`
	SelfMMR      int      `json:"selfMMR"`
	OppMMR       int      `json:"oppMMR"`
	SelfRD       float64  `json:"selfRD"`
	OppRD        float64  `json:"oppRD"`
	Winner       string   `json:"winner"` // "self" | "opp" | "draw"
	SelfIdleDays *float64 `json:"selfIdleDays"`
	OppIdleDays  *float64 `json:"oppIdleDays"`
}

type parityResult struct {
	ID        string  `json:"id"`
	SelfDelta int     `json:"selfDelta"`
	OppDelta  int     `json:"oppDelta"`
	SelfRD    float64 `json:"selfRD"`
	OppRD     float64 `json:"oppRD"`
}

type parityFile struct {
	FixedNow string         `json:"fixedNow"`
	Cases    []parityCase   `json:"cases"`
	Results  []parityResult `json:"results"`
}

func loadParityInputs(t *testing.T) (time.Time, []parityCase) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(sharedDir, "rating-cases.json"))
	if err != nil {
		t.Fatalf("read shared rating cases: %v", err)
	}
	var parsed struct {
		FixedNow string       `json:"fixedNow"`
		Cases    []parityCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parse shared rating cases: %v", err)
	}
	fixedNow, err := time.Parse(time.RFC3339, parsed.FixedNow)
	if err != nil {
		t.Fatalf("parse fixedNow: %v", err)
	}
	if len(parsed.Cases) == 0 {
		t.Fatal("shared rating case corpus is empty")
	}
	return fixedNow, parsed.Cases
}

func updatedAt(fixedNow time.Time, idleDays *float64) time.Time {
	if idleDays == nil {
		return time.Time{}
	}
	return fixedNow.Add(time.Duration(-*idleDays * float64(24*time.Hour)))
}

func computeParityResults(fixedNow time.Time, cases []parityCase) []parityResult {
	results := make([]parityResult, 0, len(cases))
	for _, c := range cases {
		self := rating.State{MMR: c.SelfMMR, RD: c.SelfRD, UpdatedAt: updatedAt(fixedNow, c.SelfIdleDays)}
		opp := rating.State{MMR: c.OppMMR, RD: c.OppRD, UpdatedAt: updatedAt(fixedNow, c.OppIdleDays)}
		winner := c.Winner
		switch winner {
		case "self":
			winner = "p1"
		case "opp":
			winner = "p2"
		case "draw":
			winner = ""
		}
		selfUpdate, oppUpdate := rating.CalculateDuelUpdates(self, opp, winner, fixedNow)
		results = append(results, parityResult{
			ID:        c.ID,
			SelfDelta: selfUpdate.Delta,
			OppDelta:  oppUpdate.Delta,
			SelfRD:    selfUpdate.RD,
			OppRD:     oppUpdate.RD,
		})
	}
	return results
}

func TestRatingGoldenParity(t *testing.T) {
	fixedNow, cases := loadParityInputs(t)
	results := computeParityResults(fixedNow, cases)

	goldenPath := filepath.Join(sharedDir, "rating-expected.json")
	if os.Getenv("GEODUELS_UPDATE_RATING_GOLDEN") == "1" {
		out, err := json.MarshalIndent(parityFile{FixedNow: fixedNow.Format(time.RFC3339), Cases: cases, Results: results}, "", "  ")
		if err != nil {
			t.Fatalf("marshal golden: %v", err)
		}
		if err := os.WriteFile(goldenPath, append(out, '\n'), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Log("rating golden updated")
		return
	}

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read rating golden (regenerate with GEODUELS_UPDATE_RATING_GOLDEN=1 go test ./internal/rating): %v", err)
	}
	var golden parityFile
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatalf("parse rating golden: %v", err)
	}
	if len(golden.Results) != len(results) {
		t.Fatalf("golden has %d results, computed %d; regenerate the golden file", len(golden.Results), len(results))
	}
	for i, want := range golden.Results {
		got := results[i]
		if got.ID != want.ID {
			t.Fatalf("case %d: id mismatch %q vs %q; regenerate the golden file", i, got.ID, want.ID)
		}
		if got.SelfDelta != want.SelfDelta || got.OppDelta != want.OppDelta {
			t.Fatalf("case %s: deltas drifted: go=(%d,%d) golden=(%d,%d)", got.ID, got.SelfDelta, got.OppDelta, want.SelfDelta, want.OppDelta)
		}
		if diff := abs(got.SelfRD - want.SelfRD); diff > 1e-9 {
			t.Fatalf("case %s: self RD drifted: go=%v golden=%v", got.ID, got.SelfRD, want.SelfRD)
		}
		if diff := abs(got.OppRD - want.OppRD); diff > 1e-9 {
			t.Fatalf("case %s: opp RD drifted: go=%v golden=%v", got.ID, got.OppRD, want.OppRD)
		}
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
