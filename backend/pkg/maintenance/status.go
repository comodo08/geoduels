package maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const RedisKey = "system:maintenance"

type Phase string

const (
	PhaseNormal  Phase = "normal"
	PhaseWarning Phase = "warning"
	PhaseActive  Phase = "active"
)

type Status struct {
	Phase       Phase      `json:"phase"`
	StartsAt    *time.Time `json:"startsAt,omitempty"`
	EndsAt      *time.Time `json:"endsAt,omitempty"`
	QueuePaused bool       `json:"queuePaused,omitempty"`
	PlayPaused  bool       `json:"playPaused,omitempty"`
	Message     string     `json:"message,omitempty"`
}

func DefaultStatus() Status {
	return Status{Phase: PhaseNormal}
}

func (s Status) Normalized() Status {
	switch s.Phase {
	case PhaseNormal, PhaseWarning, PhaseActive:
	default:
		s.Phase = PhaseNormal
	}
	s.Message = strings.TrimSpace(s.Message)
	return s
}

func (s Status) IsVisible() bool {
	return s.Phase != PhaseNormal || s.QueuePaused || s.PlayPaused || s.Message != "" || s.StartsAt != nil || s.EndsAt != nil
}

func (s Status) QueueBlocked() bool {
	return s.QueuePaused || s.PlayPaused
}

func (s Status) PlayBlocked() bool {
	return s.PlayPaused
}

func Read(ctx context.Context, rdb *redis.Client) (Status, error) {
	return ReadKey(ctx, rdb, RedisKey)
}

func ReadKey(ctx context.Context, rdb *redis.Client, key string) (Status, error) {
	if rdb == nil {
		return DefaultStatus(), nil
	}
	raw, err := rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return DefaultStatus(), nil
		}
		return DefaultStatus(), err
	}
	var status Status
	if err := json.Unmarshal([]byte(raw), &status); err != nil {
		return DefaultStatus(), err
	}
	return status.Normalized(), nil
}
