package preferences

import (
	"context"
	"encoding/json"
)

// Store is the persistence capability required by the preferences use cases.
type Store interface {
	GetUserPreferences(context.Context, string) (UserPreferences, error)
	UpdateUserPreferences(context.Context, string, int, json.RawMessage, int64) (UserPreferences, error)
}

type Service struct{ store Store }

func NewService(store Store) *Service { return &Service{store: store} }

func (s *Service) Get(ctx context.Context, userID string) (UserPreferences, error) {
	return s.store.GetUserPreferences(ctx, userID)
}

func (s *Service) Update(ctx context.Context, userID string, version int, value json.RawMessage, revision int64) (UserPreferences, error) {
	if err := ValidateVersion(version); err != nil {
		return UserPreferences{}, err
	}
	return s.store.UpdateUserPreferences(ctx, userID, version, value, revision)
}
