package storage

import "time"

type Store interface {
	CleanupStorage(batchSize int) (StorageCleanupResult, error)
	ReconcileStaleMatchSessions(grace time.Duration, batchSize int) (int64, error)
}

type Service struct{ Store }

func NewService(store Store) *Service { return &Service{Store: store} }
