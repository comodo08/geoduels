package social

import (
	"context"
	"time"
)

// Store persists social data. WithinTx supplies a store bound to one
// transaction; callback errors roll back all its writes.
type Store interface {
	WithinTx(ctx context.Context, fn func(Store) error) error
	GetSocialAccount(ctx context.Context, userID string) (SocialAccount, error)
	GetSocialSettings(ctx context.Context, userID string) (SocialSettings, error)
	UpdateSocialSettings(ctx context.Context, userID string, settings SocialSettings) (SocialSettings, error)
	Relationship(ctx context.Context, userID, targetID string) (RelationshipState, string, error)
	ListFriends(ctx context.Context, userID string, limit int) ([]CompactPlayer, error)
	ListFriendRequests(ctx context.Context, userID, direction string, limit int) ([]FriendRequest, error)
	SearchSocialPlayers(ctx context.Context, userID, query string, limit int) ([]CompactPlayer, error)
	ListRecentPlayers(ctx context.Context, userID string, limit int) ([]CompactPlayer, error)
	CountFriends(ctx context.Context, userID string) (int, error)
	SendAllowed(ctx context.Context, userID, targetID string) (bool, error)
	CrossedRequest(ctx context.Context, senderID, recipientID string) (string, bool, error)
	InsertFriendRequest(ctx context.Context, senderID, recipientID string, expiresAt time.Time) (FriendRequest, error)
	FriendRequestSender(ctx context.Context, requestID, recipientID string) (string, error)
	AcceptFriendRequest(ctx context.Context, requestID, recipientID string) error
	DeclineFriendRequest(ctx context.Context, requestID, recipientID string) error
	CancelFriendRequest(ctx context.Context, requestID, senderID string) error
	MarkFriendRequestNotificationRead(ctx context.Context, requestID string) error
	RemoveFriend(ctx context.Context, userID, targetID string) error
	AddUserBlock(ctx context.Context, userID, targetID string) error
	RemoveUserBlock(ctx context.Context, userID, targetID string) error
	CancelPairFriendRequests(ctx context.Context, userID, targetID string) error
	NotifyUser(ctx context.Context, userID, notificationType, dedupeKey string, payload map[string]any, actorID string) error
	RevokeFriendCodes(ctx context.Context, userID string) error
	InsertFriendCode(ctx context.Context, userID, code string, expiresAt time.Time) error
	ResolveFriendCode(ctx context.Context, userID, code string) (CompactPlayer, error)
	InvitationEligibility(ctx context.Context, partyID, inviterID, recipientID string) (PartyInvitation, error)
	PendingPartyInvitation(ctx context.Context, partyID, recipientID string) (PartyInvitation, bool, error)
	UpsertPartyInvitation(ctx context.Context, partyID, inviterID, recipientID string, expiresAt, createdAt time.Time) (PartyInvitation, error)
	ListPartyInvitations(ctx context.Context, userID string, limit int) ([]PartyInvitation, error)
	ListPartyInviteStatus(ctx context.Context, inviterID, partyID string) (map[string]CompactPartyInvite, error)
	RespondPartyInvitation(ctx context.Context, userID, invitationID, response string) (PartyInvitation, error)
	MarkPartyInvitationNotificationRead(ctx context.Context, userID, invitationID string) error
}
