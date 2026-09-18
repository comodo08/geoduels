package matches

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"geoduels/pkg/persistence"
)

func TestReplayCompressionRoundTrip(t *testing.T) {
	raw := bytes.Repeat([]byte(`{"matchId":"match","players":{"u":{"score":5000}}}`), 100)
	compressed, sum, err := CompressReplay(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(compressed) >= len(raw) {
		t.Fatalf("compressed replay size=%d, raw=%d", len(compressed), len(raw))
	}
	decoded, err := decompressReplay(compressed, ReplayCodecZstd, len(raw))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded, raw) {
		t.Fatal("decoded replay differs from original")
	}
	if len(sum) != 32 {
		t.Fatalf("sha256 length=%d", len(sum))
	}
}

func TestReplayDecompressionRejectsInvalidMetadata(t *testing.T) {
	if _, err := decompressReplay([]byte("bad"), 99, 3); err == nil {
		t.Fatal("expected unsupported codec error")
	}
	if _, err := decompressReplay([]byte("bad"), ReplayCodecZstd, maxReplayDecodedBytes+1); err == nil {
		t.Fatal("expected decoded size limit error")
	}
}

func TestReplayReadIntegration(t *testing.T) {
	dsn := os.Getenv("REPLAY_TEST_POSTGRES_URL")
	matchID := os.Getenv("REPLAY_TEST_MATCH_ID")
	if dsn == "" || matchID == "" {
		t.Skip("REPLAY_TEST_POSTGRES_URL and REPLAY_TEST_MATCH_ID are required")
	}
	t.Setenv("POSTGRES_URL", dsn)
	dbstore, err := persistence.NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	defer dbstore.Close()
	store := NewPGStore(dbstore.Pool())
	raw, found, err := store.GetFinalMatchSnapshot(matchID)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("compressed replay not found")
	}
	if !json.Valid(raw) {
		t.Fatal("decoded replay is not valid JSON")
	}
}
