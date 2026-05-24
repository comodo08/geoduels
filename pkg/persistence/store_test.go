package persistence

import (
	"os"
	"strings"
	"testing"
)

func TestRecordMatchResultFinalRankedDeltaCastsParameters(t *testing.T) {
	body, err := os.ReadFile("store.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(body)
	if strings.Contains(source, "final_ranked_delta = $3 - $2") {
		t.Fatal("final_ranked_delta must cast pgx parameters before subtraction")
	}
	if got := strings.Count(source, "final_ranked_delta = $3::integer - $2::integer"); got != 2 {
		t.Fatalf("typed final_ranked_delta expressions = %d, want 2", got)
	}
}
