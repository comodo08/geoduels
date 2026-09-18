package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type SessionRecord struct {
	UserID     string `json:"userId"`
	ConnID     string `json:"connId"`
	InstanceID string `json:"instanceId"`
	MatchID    string `json:"matchId,omitempty"`
	UpdatedAt  int64  `json:"updatedAt"`
}

type SessionStore interface {
	BindConn(ctx context.Context, userID, connID, instanceID string) error
	BindMatch(ctx context.Context, userID, matchID string) error
	UnbindMatch(ctx context.Context, userID string) error
	ResolveUser(ctx context.Context, userID string) (SessionRecord, bool, error)
	UnbindConn(ctx context.Context, userID, connID string) error
}

type RedisSessionStore struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewRedisSessionStore(rdb *redis.Client, ttl time.Duration) SessionStore {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return &RedisSessionStore{rdb: rdb, ttl: ttl}
}

func (s *RedisSessionStore) BindConn(ctx context.Context, userID, connID, instanceID string) error {
	if userID == "" || connID == "" || instanceID == "" {
		return errors.New("userID, connID, instanceID required")
	}
	matchID := ""
	if prev, ok, err := s.ResolveUser(ctx, userID); err == nil && ok {
		matchID = prev.MatchID
	}
	rec := SessionRecord{
		UserID:     userID,
		ConnID:     connID,
		InstanceID: instanceID,
		MatchID:    matchID,
		UpdatedAt:  time.Now().UnixMilli(),
	}
	return s.set(ctx, rec)
}

func (s *RedisSessionStore) BindMatch(ctx context.Context, userID, matchID string) error {
	rec, ok, err := s.ResolveUser(ctx, userID)
	if err != nil {
		return err
	}
	if !ok {
		rec = SessionRecord{UserID: userID}
	}
	rec.MatchID = matchID
	rec.UpdatedAt = time.Now().UnixMilli()
	return s.set(ctx, rec)
}

func (s *RedisSessionStore) UnbindMatch(ctx context.Context, userID string) error {
	rec, ok, err := s.ResolveUser(ctx, userID)
	if err != nil || !ok {
		return err
	}
	rec.MatchID = ""
	rec.UpdatedAt = time.Now().UnixMilli()
	return s.set(ctx, rec)
}

func (s *RedisSessionStore) ResolveUser(ctx context.Context, userID string) (SessionRecord, bool, error) {
	b, err := s.rdb.Get(ctx, sessionKey(userID)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return SessionRecord{}, false, nil
		}
		return SessionRecord{}, false, err
	}
	var rec SessionRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		return SessionRecord{}, false, err
	}
	return rec, true, nil
}

func (s *RedisSessionStore) UnbindConn(ctx context.Context, userID, connID string) error {
	rec, ok, err := s.ResolveUser(ctx, userID)
	if err != nil || !ok {
		return err
	}
	if rec.ConnID != connID {
		return nil
	}
	rec.ConnID = ""
	rec.UpdatedAt = time.Now().UnixMilli()
	return s.set(ctx, rec)
}

func (s *RedisSessionStore) set(ctx context.Context, rec SessionRecord) error {
	b, _ := json.Marshal(rec)
	return s.rdb.Set(ctx, sessionKey(rec.UserID), b, s.ttl).Err()
}

func sessionKey(userID string) string {
	return "rt:user:session:" + userID
}
