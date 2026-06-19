package persistence

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *pgStore) GetFinalMatchSnapshot(matchID string) ([]byte, bool, error) {
	if matchID == "" {
		return nil, false, errors.New("matchID required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	row := s.pool.QueryRow(ctx, `
		select replay_zstd, coalesce(replay_codec, 0), coalesce(replay_uncompressed_bytes, 0),
		       replay_sha256, replay_json::text
		from match_history
		where match_id = $1
		  and (replay_expires_at is null or replay_expires_at > now())
		limit 1
	`, matchID)
	var compressed, expectedHash []byte
	var codec, uncompressedBytes int
	var legacy *string
	if err := row.Scan(&compressed, &codec, &uncompressedBytes, &expectedHash, &legacy); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if len(compressed) == 0 {
		if legacy == nil {
			return nil, false, nil
		}
		return []byte(*legacy), true, nil
	}
	raw, err := decompressReplay(compressed, codec, uncompressedBytes)
	if err != nil {
		return nil, false, err
	}
	if len(expectedHash) == sha256.Size {
		sum := sha256.Sum256(raw)
		if !equalBytes(sum[:], expectedHash) {
			return nil, false, errors.New("replay checksum mismatch")
		}
	}
	return raw, true, nil
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}

func (s *pgStore) ListPlayerMatchHistory(userID string, limit int) ([]MatchHistorySummary, error) {
	if userID == "" {
		return nil, errors.New("userID required")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
		select h.match_id, h.mode, h.started_at, h.ended_at, coalesce(h.winner_user_id::text, '')
		from match_history h
		join match_players p on p.match_id = h.match_id
		where p.user_id = $1
		order by h.ended_at desc, h.match_id desc
		limit $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]MatchHistorySummary, 0, limit)
	for rows.Next() {
		var item MatchHistorySummary
		if err := rows.Scan(&item.MatchID, &item.Mode, &item.StartedAt, &item.EndedAt, &item.WinnerUserID); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
