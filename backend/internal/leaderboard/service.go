package leaderboard

import "context"

type Store interface {
	ListLeaderboard(context.Context, string, string, int, int) ([]Entry, error)
	GetLeaderboardOverview(context.Context, string, string, string, int) (Overview, error)
}

type Service struct{ store Store }

func NewService(store Store) *Service { return &Service{store: store} }
func (s *Service) List(ctx context.Context, mode, season string, limit, offset int) ([]Entry, error) {
	return s.store.ListLeaderboard(ctx, mode, season, limit, offset)
}
func (s *Service) Overview(ctx context.Context, userID, mode, season string, limit int) (Overview, error) {
	return s.store.GetLeaderboardOverview(ctx, userID, mode, season, limit)
}
