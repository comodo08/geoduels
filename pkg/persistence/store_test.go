package persistence

import (
	"os"
	"strings"
	"testing"
)

func TestFinalizeMatchFinalRankedDeltaCastsParameters(t *testing.T) {
	body, err := os.ReadFile("matches_write.go")
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

func TestRecordMatchHistoryCastsReplayExpirationParameters(t *testing.T) {
	body, err := os.ReadFile("matches_write.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(body)
	if strings.Contains(source, "$4 + make_interval(days => $16)") {
		t.Fatal("replay expiration must cast pgx timestamp and integer parameters")
	}
	if !strings.Contains(source, "$4::timestamptz + make_interval(days => $16::integer)") {
		t.Fatal("typed replay expiration expression is missing")
	}
}

func TestModerationProjectionCastsUUIDBeforeAggregate(t *testing.T) {
	body, err := os.ReadFile("moderation_scoring.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(body)
	if strings.Contains(source, "max(reported_user_id)::text") {
		t.Fatal("PostgreSQL cannot aggregate UUID values with max before casting")
	}
	if !strings.Contains(source, "max(reported_user_id::text)") {
		t.Fatal("moderation projection must cast UUID values before max")
	}
}

func TestCheatingBanRefundQueryUsesCompactRatingColumns(t *testing.T) {
	body, err := os.ReadFile("moderation_enforcement.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(body)
	if strings.Contains(source, "snapshot_json->'ratingPreview'") {
		t.Fatal("cheating-ban refund query must not depend on retained replay JSON")
	}
	if !strings.Contains(source, "opponent.final_ranked_delta as original_delta") {
		t.Fatal("cheating-ban refund query must use compact persisted rating deltas")
	}
}
