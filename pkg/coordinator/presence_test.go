package coordinator

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestGetPresenceStatusesBucketsByAge(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	store := NewStore(rdb, 10*time.Second, 2*time.Hour, 24*time.Hour, 5*time.Second)
	ctx := context.Background()

	now := time.Now().UnixMilli()
	// u_online: touched just now.
	// u_away: touched 30s ago (within away window, outside online window).
	// u_offline: touched 90s ago (outside away window).
	// u_absent: never present.
	if err := rdb.ZAdd(ctx, presenceKey(), redis.Z{Score: float64(now), Member: "u_online"}).Err(); err != nil {
		t.Fatalf("seed online: %v", err)
	}
	if err := rdb.ZAdd(ctx, presenceKey(), redis.Z{Score: float64(now - 30*1000), Member: "u_away"}).Err(); err != nil {
		t.Fatalf("seed away: %v", err)
	}
	if err := rdb.ZAdd(ctx, presenceKey(), redis.Z{Score: float64(now - 90*1000), Member: "u_offline"}).Err(); err != nil {
		t.Fatalf("seed offline: %v", err)
	}

	statuses, err := store.GetPresenceStatuses(ctx, []string{"u_online", "u_away", "u_offline", "u_absent"})
	if err != nil {
		t.Fatalf("GetPresenceStatuses: %v", err)
	}
	if statuses["u_online"] != "online" {
		t.Fatalf("u_online = %q, want online", statuses["u_online"])
	}
	if statuses["u_away"] != "away" {
		t.Fatalf("u_away = %q, want away", statuses["u_away"])
	}
	if statuses["u_offline"] != "offline" {
		t.Fatalf("u_offline = %q, want offline", statuses["u_offline"])
	}
	if statuses["u_absent"] != "offline" {
		t.Fatalf("u_absent = %q, want offline", statuses["u_absent"])
	}
}

func TestGetPresenceStatusesEmptyInput(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	store := NewStore(rdb, 10*time.Second, 2*time.Hour, 24*time.Hour, 5*time.Second)
	ctx := context.Background()

	statuses, err := store.GetPresenceStatuses(ctx, nil)
	if err != nil {
		t.Fatalf("GetPresenceStatuses(nil): %v", err)
	}
	if len(statuses) != 0 {
		t.Fatalf("expected empty map, got %v", statuses)
	}
}
