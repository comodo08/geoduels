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

func TestCheatingBanRefundQueryCastsOpponentRatingPreviewKey(t *testing.T) {
	body, err := os.ReadFile("store.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(body)
	if strings.Contains(source, "->(::text)opponent_user_id") {
		t.Fatal("cheating-ban refund query must cast opponent_user_id before JSONB key lookup")
	}
	if !strings.Contains(source, "snapshot_json->'ratingPreview'->(opponent_user_id::text)->>'lose'") {
		t.Fatal("cheating-ban refund query must read opponent rating preview using opponent_user_id as a JSONB key")
	}
}
