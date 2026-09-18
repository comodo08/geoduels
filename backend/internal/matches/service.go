package matches

import (
	"context"
	"time"

	"geoduels/pkg/contracts"
)

type Store interface {
	FinalizeMatch(snap contracts.MatchSnapshot, ownerEpoch int64) (contracts.MatchSnapshot, error)
	RenewMatchSessionLeases(nodeID string, ownerEpoch int64, matchIDs []string, ttl time.Duration) error
	GetFinalMatchSnapshot(matchID string) ([]byte, bool, error)
	ListPlayerMatchHistory(userID string, limit int) ([]MatchHistorySummary, error)
	ListPlayerMatchHistoryPage(userID string, limit int, beforeEndedAt time.Time, beforeMatchID string, rankedOnly bool) (MatchHistoryPage, error)
	PlayerParticipatedInMatch(userID, matchID string) (bool, error)
	GetRuntimeMatch(ctx context.Context, matchID string) (RuntimeMatch, bool, error)
	RecordRuntimeMatch(ctx context.Context, matchID, state string, ownerEpoch int64, terminal bool) error
	UpsertMatchSession(ctx context.Context, params MatchSessionUpsert) error
	MatchSessionSourceParty(ctx context.Context, matchID string) (string, string, bool, error)
	MatchSessionReturnTarget(ctx context.Context, matchID string) (*contracts.MatchReturnTarget, bool, error)
	ExpireStaleRuntimeMatches(ctx context.Context, prefix string, olderThan time.Duration) error
}

type Service struct{ Store }

func NewService(store Store) *Service { return &Service{Store: store} }
