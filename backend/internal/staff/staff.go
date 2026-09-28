// Package staff implements staff authorization, policies and operations.
package staff

import (
	"context"
	"errors"
	"time"

	"geoduels/pkg/contracts"
	pkgstaff "geoduels/pkg/staff"
)

// Actor is the authenticated staff principal.
type Actor = pkgstaff.Actor

// ErrForbidden is returned when the actor lacks the required capability.
var ErrForbidden = pkgstaff.ErrForbidden

// ErrUnavailable marks a target (map, nomination, player) that is not eligible.
var ErrUnavailable = errors.New("target unavailable")

// ErrNotFound marks a missing subject.
var ErrNotFound = errors.New("subject not found")

// Action identifies an audited staff action. Values match the moderation_log
// action enum so one audit writer persists every action.
type Action string

const (
	ActionRoleGranted  Action = "role_granted"
	ActionRoleRevoked  Action = "role_revoked"
	ActionBadgeGranted Action = "badge_granted"
	ActionPermanentBan Action = "permanent_ban"
	ActionTemporaryBan Action = "temporary_ban"
	ActionChatMute     Action = "chat_mute"
	ActionReportMute   Action = "report_mute"
	ActionReportUnmute Action = "report_unmute"
	ActionUnban        Action = "unban"
	ActionRefund       Action = "refund"
	ActionNote         Action = "note"
)

// AuditEntry is one record on the unified staff audit spindle.
type AuditEntry struct {
	ActorID   string
	SubjectID string
	Action    Action
	Reason    string
	ExpiresAt *time.Time
	Metadata  any
	SignalIDs []int64
}

// SubjectDetail is a player read shape already filtered for the viewer's role.
type SubjectDetail struct {
	Player contracts.AdminPlayerSummary `json:"player"`
}

// Service applies staff policies and coordinates persistence. The detector is
// an external dependency, independent of database transactions.
type Service struct {
	store Store
	risk  RiskEngine
	clock func() time.Time
}

func NewService(store Store, risk RiskEngine) *Service {
	return &Service{store: store, risk: risk, clock: time.Now}
}

func (s *Service) now() time.Time { return s.clock().UTC() }

// require checks the actor holds one of the capabilities (read paths).
func (s *Service) require(actor Actor, caps ...pkgstaff.Capability) error {
	for _, cap := range caps {
		if actor.Can(cap) {
			return nil
		}
	}
	return pkgstaff.ErrForbidden
}

func requireCap(cap pkgstaff.Capability) func(Actor) error {
	return func(actor Actor) error { return actor.RequireCap(cap) }
}

func requireAny(caps ...pkgstaff.Capability) func(Actor) error {
	return func(actor Actor) error {
		for _, cap := range caps {
			if actor.Can(cap) {
				return nil
			}
		}
		return pkgstaff.ErrForbidden
	}
}

// run authorizes then executes fn with a transaction-bound store.
func run[R any](ctx context.Context, store Store, actor Actor, authorize func(Actor) error, fn func(ctx context.Context, actor Actor, tx Store) (R, error)) (R, error) {
	var zero R
	if err := authorize(actor); err != nil {
		return zero, err
	}
	var out R
	err := store.WithinTx(ctx, func(tx Store) error {
		result, err := fn(ctx, actor, tx)
		if err != nil {
			return err
		}
		out = result
		return nil
	})
	if err != nil {
		return zero, err
	}
	return out, nil
}

func runVoid(ctx context.Context, store Store, actor Actor, authorize func(Actor) error, fn func(ctx context.Context, actor Actor, tx Store) error) error {
	_, err := run(ctx, store, actor, authorize, func(ctx context.Context, actor Actor, tx Store) (struct{}, error) {
		return struct{}{}, fn(ctx, actor, tx)
	})
	return err
}
