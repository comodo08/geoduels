package social

import "time"

// Social rules that callers and the store must agree on live here so the
// limits are not duplicated between handlers and SQL workflows.

const (
	// FriendRequestTTL is how long a friend request stays pending.
	FriendRequestTTL = 30 * 24 * time.Hour
	// FriendLimit is the maximum number of friends an account may have.
	FriendLimit = 500
	// DefaultFriendCodeTTL is used when a friend code is created without a TTL.
	DefaultFriendCodeTTL = 7 * 24 * time.Hour
	// DefaultPartyInviteTTL is used when a party invitation is created without a TTL.
	DefaultPartyInviteTTL = 20 * time.Minute
	// PartyInviteResendAfter is the resend cooldown for an existing pending invitation.
	PartyInviteResendAfter = 30 * time.Second

	// List limits used by the read use cases.
	FriendsListLimit    = 100
	FriendRequestsLimit = 20
	RecentPlayersLimit  = 3
	SearchLimit         = 10
	SearchMaxLimit      = 20
)

type Account struct {
	IsGuest       bool
	ActionEnabled bool
	Blocked       bool
	TargetExists  bool
	AtLimit       bool
	SameUser      bool
}

// Authorize is the common policy gate for relationship, code, invitation, and
// discovery actions. Persistence still rechecks relational invariants inside
// the mutation transaction to prevent races. It maps onto the service errors:
// guests get ErrRegistrationRequired, unavailable actions ErrBlocked, and
// exhausted budgets ErrLimit.
func Authorize(account Account) error {
	switch {
	case account.IsGuest:
		return ErrRegistrationRequired
	case account.SameUser, account.Blocked, !account.TargetExists, !account.ActionEnabled:
		return ErrBlocked
	case account.AtLimit:
		return ErrLimit
	default:
		return nil
	}
}

// BoundedLimit clamps a caller-provided list size to the feature's bounds.
func BoundedLimit(limit, fallback, maximum int) int {
	if limit <= 0 {
		return fallback
	}
	if limit > maximum {
		return maximum
	}
	return limit
}
