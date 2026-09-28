package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"geoduels/pkg/maintenance"
)

func TestMaintenanceDrainTriggers(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status maintenance.Status
		drain  bool
	}{
		{"normal", maintenance.DefaultStatus(), false},
		{"warning", maintenance.Status{Phase: maintenance.PhaseWarning}, false},
		{"queue only", maintenance.Status{QueuePaused: true}, false},
		{"active", maintenance.Status{Phase: maintenance.PhaseActive}, true},
		{"play paused", maintenance.Status{Phase: maintenance.PhaseWarning, PlayPaused: true}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var drain gameplayDrain
			drain.setMaintenance(tc.status, time.Now())
			if drain.isDraining() != tc.drain {
				t.Fatalf("draining = %v, want %v", drain.isDraining(), tc.drain)
			}
			if drain.isStopping() {
				t.Fatal("maintenance must not stop the process or block rollout readiness")
			}
		})
	}
}

func TestOrderedRolloutSharesMaintenanceDrainBudget(t *testing.T) {
	maintenanceAt := time.Date(2026, 9, 28, 11, 39, 0, 0, time.UTC)
	status := maintenance.Status{Phase: maintenance.PhaseActive}
	var nodes [2]gameplayDrain
	for i := range nodes {
		nodes[i].setMaintenance(status, maintenanceAt)
		// Repeated polling or edits to the announcement must not reset the timer.
		nodes[i].setMaintenance(status, maintenanceAt.Add(10*time.Minute))
	}
	for i := range nodes {
		stopAt := maintenanceAt.Add(time.Duration(16+i) * time.Minute)
		startedAt := nodes[i].startShutdown(stopAt)
		if !startedAt.Equal(maintenanceAt) || startedAt.Add(15*time.Minute).After(stopAt) {
			t.Fatalf("node %d received a fresh drain budget: %v", i, startedAt)
		}
		// Clearing maintenance while Kubernetes is stopping a pod cannot undo it.
		nodes[i].setMaintenance(maintenance.DefaultStatus(), stopAt)
		if !nodes[i].isDraining() || !nodes[i].isStopping() {
			t.Fatal("maintenance cancellation undid shutdown")
		}
	}
}

func TestMaintenanceCancellationResetsDrainBudget(t *testing.T) {
	var drain gameplayDrain
	start := time.Now()
	drain.setMaintenance(maintenance.Status{PlayPaused: true}, start)
	drain.setMaintenance(maintenance.DefaultStatus(), start.Add(time.Minute))
	if drain.isDraining() || drain.isStopping() {
		t.Fatal("clearing maintenance should reopen match admission")
	}
	next := start.Add(time.Hour)
	drain.setMaintenance(maintenance.Status{PlayPaused: true}, next)
	if got := drain.startShutdown(next.Add(time.Minute)); !got.Equal(next) {
		t.Fatalf("new maintenance reused old drain time: %v", got)
	}
}

func TestShutdownWithoutMaintenanceStillDrains(t *testing.T) {
	var drain gameplayDrain
	now := time.Now()
	if got := drain.startShutdown(now); !got.Equal(now) {
		t.Fatalf("shutdown drain started at %v, want %v", got, now)
	}
	if !drain.isDraining() || !drain.isStopping() {
		t.Fatal("shutdown must stop admission and readiness")
	}
}

func TestMaintenanceDrainsAdmissionButKeepsRolloutReady(t *testing.T) {
	hook := &maintenanceRedisHook{value: `{"phase":"active"}`}
	rdb := redis.NewClient(&redis.Options{})
	t.Cleanup(func() { _ = rdb.Close() })
	rdb.AddHook(hook)
	g := gameplayNode{redis: rdb, coordAuth: "test-secret"}
	g.refreshMaintenanceDrain()

	e := echo.New()
	request := httptest.NewRequest(http.MethodPost, "/internal/matches", nil)
	request.Header.Set("X-Coordinator-Secret", g.coordAuth)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	if err := g.createMatch(ctx); err != nil {
		e.HTTPErrorHandler(err, ctx)
	}
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("maintenance admitted new match: HTTP %d", recorder.Code)
	}
	checkReady := func(want int) {
		t.Helper()
		recorder := httptest.NewRecorder()
		ctx := e.NewContext(httptest.NewRequest(http.MethodGet, "/health/ready", nil), recorder)
		if err := g.healthReady(ctx); err != nil {
			e.HTTPErrorHandler(err, ctx)
		}
		if recorder.Code != want {
			t.Fatalf("readiness = %d, want %d", recorder.Code, want)
		}
	}
	checkReady(http.StatusOK)
	hook.err = errors.New("Redis unavailable")
	g.refreshMaintenanceDrain()
	if !g.drain.isDraining() {
		t.Fatal("Redis failure cleared maintenance drain")
	}
	hook.err = nil
	g.drain.startShutdown(time.Now())
	checkReady(http.StatusServiceUnavailable)
}

// Answer Redis commands in memory; no database or network is used.
type maintenanceRedisHook struct {
	value string
	err   error
}

func (h *maintenanceRedisHook) DialHook(redis.DialHook) redis.DialHook {
	return func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("unexpected Redis connection")
	}
}

func (h *maintenanceRedisHook) ProcessHook(redis.ProcessHook) redis.ProcessHook {
	return func(_ context.Context, cmd redis.Cmder) error {
		if h.err != nil {
			return h.err
		}
		switch cmd := cmd.(type) {
		case *redis.StringCmd:
			cmd.SetVal(h.value)
		case *redis.StatusCmd:
			cmd.SetVal("PONG")
		default:
			return errors.New("unexpected Redis command")
		}
		return nil
	}
}

func (h *maintenanceRedisHook) ProcessPipelineHook(redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(context.Context, []redis.Cmder) error {
		return errors.New("unexpected Redis pipeline")
	}
}
