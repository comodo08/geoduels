package staff

import (
	"context"
	"encoding/json"

	"geoduels/pkg/maintenance"
)

func (a *PGStore) GetMaintenance(ctx context.Context) (maintenance.Status, error) {
	return maintenance.Read(ctx, a.redis)
}

func (a *PGStore) SetMaintenance(ctx context.Context, status maintenance.Status) error {
	body, err := json.Marshal(status)
	if err != nil {
		return err
	}
	return a.redis.Set(ctx, maintenance.RedisKey, body, 0).Err()
}

func (a *PGStore) ClearMaintenance(ctx context.Context) error {
	return a.redis.Del(ctx, maintenance.RedisKey).Err()
}
