package social

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"geoduels/internal/storekit"
	"geoduels/pkg/entityid"
	db "geoduels/pkg/persistence/sqlc/db"
)

// PGStore owns PostgreSQL access for the social feature: relationships,
// settings, friend codes, and party invitations. It implements Store.
type PGStore struct {
	pool *pgxpool.Pool
}

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

func (s *PGStore) q() *db.Queries { return db.New(s.pool) }

func (s *PGStore) GetSocialSettings(ctx context.Context, userID string) (SocialSettings, error) {
	id, err := storekit.ProfileUUID(userID)
	if err != nil {
		return SocialSettings{}, err
	}
	row, err := s.q().GetSocialSettings(ctx, id)
	settings := SocialSettings{Discoverable: row.SocialDiscoverable, PresenceVisible: row.SocialPresenceVisible, RequestsEnabled: row.SocialRequestsEnabled, PartyInvitesEnabled: row.SocialPartyInvitesEnabled}
	return settings, err
}

func (s *PGStore) UpdateSocialSettings(ctx context.Context, userID string, settings SocialSettings) (SocialSettings, error) {
	id, err := storekit.ProfileUUID(userID)
	if err != nil {
		return settings, err
	}
	row, err := s.q().UpdateSocialSettings(ctx, db.UpdateSocialSettingsParams{ID: id, SocialDiscoverable: settings.Discoverable, SocialPresenceVisible: settings.PresenceVisible, SocialRequestsEnabled: settings.RequestsEnabled, SocialPartyInvitesEnabled: settings.PartyInvitesEnabled})
	settings = SocialSettings{Discoverable: row.SocialDiscoverable, PresenceVisible: row.SocialPresenceVisible, RequestsEnabled: row.SocialRequestsEnabled, PartyInvitesEnabled: row.SocialPartyInvitesEnabled}
	return settings, err
}

func (s *PGStore) GetSocialAccount(ctx context.Context, userID string) (bool, bool, bool, error) {
	id, err := storekit.ProfileUUID(userID)
	if err != nil {
		return false, false, false, err
	}
	row, err := s.q().GetSocialAccount(ctx, id)
	return string(row.AccountType) == "guest", row.SocialRequestsEnabled, row.SocialPartyInvitesEnabled, err
}

func (s *PGStore) Relationship(ctx context.Context, userID, targetID string) (RelationshipState, string, error) {
	if userID == targetID {
		return RelationshipNone, "", nil
	}
	viewerUUID, err := storekit.ProfileUUID(userID)
	if err != nil {
		return RelationshipNone, "", err
	}
	targetUUID, err := storekit.ProfileUUID(targetID)
	if err != nil {
		return RelationshipNone, "", err
	}
	row, err := db.New(s.pool).Relationship(ctx, db.RelationshipParams{BlockerUserID: viewerUUID, BlockedUserID: targetUUID})
	if err != nil {
		return RelationshipNone, "", err
	}
	if row.BlockedByViewer {
		return RelationshipBlocked, "", nil
	}
	if row.BlockedByTarget {
		return RelationshipNone, "", nil
	}
	if row.Friends {
		return RelationshipFriends, "", nil
	}
	if row.RequestID.Valid {
		requestID := storekit.UUIDVal(row.RequestID)
		if storekit.UUIDVal(row.SenderID) == userID {
			return RelationshipOutgoing, requestID, nil
		}
		return RelationshipIncoming, requestID, nil
	}
	return RelationshipNone, "", nil
}

func (s *PGStore) ListFriends(ctx context.Context, userID string, limit int) ([]CompactPlayer, error) {
	limit = BoundedLimit(limit, FriendsListLimit, 500)
	id, err := storekit.ProfileUUID(userID)
	if err != nil {
		return nil, err
	}
	seasonID, err := storekit.ActiveSeasonID(ctx, s.pool)
	if err != nil {
		return nil, err
	}
	rows, err := db.New(s.pool).ListFriends(ctx, db.ListFriendsParams{UserIDLow: id, SeasonID: seasonID, RowLimit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]CompactPlayer, 0, len(rows))
	for _, row := range rows {
		p := CompactPlayer{UserID: storekit.UUIDVal(row.UserID), DisplayName: row.DisplayName, AvatarURL: row.AvatarUrl, MMR: int(row.Mmr)}
		if row.LastSeenAt.Valid {
			value := row.LastSeenAt.Time
			p.LastSeenAt = &value
		}
		p.Relationship = RelationshipFriends
		out = append(out, p)
	}
	return out, nil
}

func (s *PGStore) ListFriendRequests(ctx context.Context, userID, direction string, limit int) ([]FriendRequest, error) {
	limit = BoundedLimit(limit, FriendRequestsLimit, 100)
	id, err := storekit.ProfileUUID(userID)
	if err != nil {
		return nil, err
	}
	seasonID, err := storekit.ActiveSeasonID(ctx, s.pool)
	if err != nil {
		return nil, err
	}
	q := db.New(s.pool)
	out := []FriendRequest{}
	if direction == "outgoing" {
		rows, err := q.ListOutgoingFriendRequests(ctx, db.ListOutgoingFriendRequestsParams{SeasonID: seasonID, SenderUserID: id, RowLimit: int32(limit)})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			item := FriendRequest{ID: storekit.UUIDVal(row.RequestID), Direction: direction, CreatedAt: row.CreatedAt.Time, ExpiresAt: row.ExpiresAt.Time}
			item.Player = CompactPlayer{UserID: storekit.UUIDVal(row.UserID), DisplayName: fmt.Sprint(row.DisplayName), AvatarURL: row.AvatarUrl, MMR: int(row.Mmr), RequestID: storekit.UUIDVal(row.RequestID), Relationship: RelationshipOutgoing}
			if row.LastSeenAt.Valid {
				value := row.LastSeenAt.Time
				item.Player.LastSeenAt = &value
			}
			out = append(out, item)
		}
	} else {
		rows, err := q.ListIncomingFriendRequests(ctx, db.ListIncomingFriendRequestsParams{SeasonID: seasonID, RecipientUserID: id, RowLimit: int32(limit)})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			item := FriendRequest{ID: storekit.UUIDVal(row.RequestID), Direction: direction, CreatedAt: row.CreatedAt.Time, ExpiresAt: row.ExpiresAt.Time}
			item.Player = CompactPlayer{UserID: storekit.UUIDVal(row.UserID), DisplayName: fmt.Sprint(row.DisplayName), AvatarURL: row.AvatarUrl, MMR: int(row.Mmr), RequestID: storekit.UUIDVal(row.RequestID), Relationship: RelationshipIncoming}
			if row.LastSeenAt.Valid {
				value := row.LastSeenAt.Time
				item.Player.LastSeenAt = &value
			}
			out = append(out, item)
		}
	}
	return out, nil
}

func (s *PGStore) SearchSocialPlayers(ctx context.Context, userID, query string, limit int) ([]CompactPlayer, error) {
	query = strings.TrimSpace(query)
	if len([]rune(query)) < 2 {
		return []CompactPlayer{}, nil
	}
	limit = BoundedLimit(limit, SearchLimit, SearchMaxLimit)
	id, err := storekit.ProfileUUID(userID)
	if err != nil {
		return nil, err
	}
	seasonID, err := storekit.ActiveSeasonID(ctx, s.pool)
	if err != nil {
		return nil, err
	}
	rows, err := db.New(s.pool).SearchSocialPlayers(ctx, db.SearchSocialPlayersParams{SeasonID: seasonID, SelfUserID: id, Query: query, RowLimit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]CompactPlayer, 0, len(rows))
	for _, row := range rows {
		p := CompactPlayer{UserID: storekit.UUIDVal(row.UserID), DisplayName: row.DisplayName, AvatarURL: row.AvatarUrl, MMR: int(row.Mmr)}
		if row.LastSeenAt.Valid {
			value := row.LastSeenAt.Time
			p.LastSeenAt = &value
		}
		p.Relationship, p.RequestID, _ = s.Relationship(ctx, userID, p.UserID)
		out = append(out, p)
	}
	return out, nil
}

func (s *PGStore) ListRecentPlayers(ctx context.Context, userID string, limit int) ([]CompactPlayer, error) {
	limit = BoundedLimit(limit, RecentPlayersLimit, RecentPlayersLimit)
	id, err := storekit.ProfileUUID(userID)
	if err != nil {
		return nil, err
	}
	seasonID, err := storekit.ActiveSeasonID(ctx, s.pool)
	if err != nil {
		return nil, err
	}
	rows, err := db.New(s.pool).ListRecentPlayers(ctx, db.ListRecentPlayersParams{SeasonID: seasonID, SelfUserID: id, RowLimit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]CompactPlayer, 0, len(rows))
	for _, row := range rows {
		p := CompactPlayer{UserID: storekit.UUIDVal(row.UserID), DisplayName: row.DisplayName, AvatarURL: row.AvatarUrl, MMR: int(row.Mmr)}
		if row.LastSeenAt.Valid {
			value := row.LastSeenAt.Time
			p.LastSeenAt = &value
		}
		if row.SharedAt.Valid {
			value := row.SharedAt.Time
			p.SharedMatchAt = &value
		}
		p.Relationship = RelationshipNone
		out = append(out, p)
	}
	return out, nil
}

func (s *PGStore) SendFriendRequest(ctx context.Context, userID, targetID string) (FriendRequest, error) {
	if userID == targetID {
		return FriendRequest{}, ErrBlocked
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return FriendRequest{}, err
	}
	defer tx.Rollback(ctx)
	userUUID, err := storekit.ProfileUUID(userID)
	if err != nil {
		return FriendRequest{}, err
	}
	targetUUID, err := storekit.ProfileUUID(targetID)
	if err != nil {
		return FriendRequest{}, err
	}
	q := db.New(tx)
	allowed, err := q.CanSendFriendRequest(ctx, db.CanSendFriendRequestParams{BlockerUserID: userUUID, ID: targetUUID})
	if err != nil || !allowed.Bool {
		return FriendRequest{}, ErrBlocked
	}
	friendCount, err := q.CountFriends(ctx, userUUID)
	if err != nil {
		return FriendRequest{}, err
	}
	if friendCount >= FriendLimit {
		return FriendRequest{}, ErrLimit
	}
	crossedID, err := q.FindCrossedFriendRequest(ctx, db.FindCrossedFriendRequestParams{RecipientUserID: userUUID, SenderUserID: targetUUID})
	if err == nil {
		crossedIDString := storekit.UUIDVal(crossedID)
		if err := acceptFriendRequestTx(ctx, tx, crossedIDString, userID); err != nil {
			return FriendRequest{}, err
		}
		return FriendRequest{ID: crossedIDString, Direction: "incoming"}, tx.Commit(ctx)
	}
	id := entityid.New()
	expiresAt := time.Now().Add(FriendRequestTTL)
	var createdAt time.Time
	requestUUID, err := storekit.ProfileUUID(id)
	if err != nil {
		return FriendRequest{}, err
	}
	row, err := q.UpsertFriendRequest(ctx, db.UpsertFriendRequestParams{ID: requestUUID, SenderUserID: userUUID, RecipientUserID: targetUUID, ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true}})
	if err != nil {
		return FriendRequest{}, err
	}
	id, createdAt, expiresAt = storekit.UUIDVal(row.ID), row.CreatedAt.Time, row.ExpiresAt.Time
	var notificationID int64
	_ = storekit.UpsertUserNotificationTx(ctx, tx, targetID, "friend_request_received", "friend_request:"+id,
		map[string]any{"requestId": id, "actorUserId": userID}, &notificationID)
	if err := tx.Commit(ctx); err != nil {
		return FriendRequest{}, err
	}
	return FriendRequest{ID: id, Direction: "outgoing", CreatedAt: createdAt, ExpiresAt: expiresAt}, nil
}

func acceptFriendRequestTx(ctx context.Context, tx pgx.Tx, requestID, recipientID string) error {
	requestUUID, err := storekit.ProfileUUID(requestID)
	if err != nil {
		return ErrNotFound
	}
	recipientUUID, err := storekit.ProfileUUID(recipientID)
	if err != nil {
		return ErrNotFound
	}
	q := db.New(tx)
	senderID, err := q.AcceptFriendRequest(ctx, db.AcceptFriendRequestParams{ID: requestUUID, RecipientUserID: recipientUUID})
	if err != nil {
		return ErrNotFound
	}
	return q.InsertFriendship(ctx, db.InsertFriendshipParams{UserA: senderID, UserB: recipientUUID, CreatedFromRequestID: requestUUID})
}

func (s *PGStore) RespondFriendRequest(ctx context.Context, userID, requestID, response string) error {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	userUUID, err := storekit.ProfileUUID(userID)
	if err != nil {
		return err
	}
	requestUUID, err := storekit.ProfileUUID(requestID)
	if err != nil {
		return ErrNotFound
	}
	q := db.New(tx)
	var otherID string
	if response == "accept" {
		otherUUID, lookupErr := q.FriendRequestSender(ctx, db.FriendRequestSenderParams{ID: requestUUID, RecipientUserID: userUUID})
		err = lookupErr
		if err != nil {
			return ErrNotFound
		}
		otherID = storekit.UUIDVal(otherUUID)
		if err := acceptFriendRequestTx(ctx, tx, requestID, userID); err != nil {
			return err
		}
		var notificationID int64
		_ = storekit.UpsertUserNotificationTx(ctx, tx, otherID, "friendship_accepted", "friendship_accepted:"+requestID,
			map[string]any{"actorUserId": userID}, &notificationID)
	} else {
		var affected int64
		if response == "cancel" {
			affected, err = q.CancelFriendRequest(ctx, db.CancelFriendRequestParams{ID: requestUUID, SenderUserID: userUUID})
		} else {
			affected, err = q.DeclineFriendRequest(ctx, db.DeclineFriendRequestParams{ID: requestUUID, RecipientUserID: userUUID})
		}
		if err != nil || affected == 0 {
			return ErrNotFound
		}
	}
	_ = q.MarkFriendRequestNotificationRead(ctx, db.MarkFriendRequestNotificationReadParams{UserID: userUUID, RequestID: storekit.IngestText(requestID)})
	return tx.Commit(ctx)
}

func (s *PGStore) RemoveFriend(ctx context.Context, userID, targetID string) error {
	userUUID, err := storekit.ProfileUUID(userID)
	if err != nil {
		return err
	}
	targetUUID, err := storekit.ProfileUUID(targetID)
	if err != nil {
		return err
	}
	return db.New(s.pool).RemoveFriend(ctx, db.RemoveFriendParams{UserA: userUUID, UserB: targetUUID})
}

func (s *PGStore) SetUserBlock(ctx context.Context, userID, targetID string, blocked bool) error {
	if userID == targetID {
		return ErrBlocked
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	userUUID, err := storekit.ProfileUUID(userID)
	if err != nil {
		return err
	}
	targetUUID, err := storekit.ProfileUUID(targetID)
	if err != nil {
		return err
	}
	q := db.New(tx)
	if blocked {
		if err := q.AddUserBlock(ctx, db.AddUserBlockParams{BlockerUserID: userUUID, BlockedUserID: targetUUID}); err != nil {
			return err
		}
		_ = q.RemoveFriend(ctx, db.RemoveFriendParams{UserA: userUUID, UserB: targetUUID})
		_ = q.CancelPairFriendRequests(ctx, db.CancelPairFriendRequestsParams{UserA: userUUID, UserB: targetUUID})
	} else {
		err = q.RemoveUserBlock(ctx, db.RemoveUserBlockParams{BlockerUserID: userUUID, BlockedUserID: targetUUID})
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *PGStore) CreateFriendCode(ctx context.Context, userID string, ttl time.Duration) (FriendCode, error) {
	if ttl <= 0 {
		ttl = DefaultFriendCodeTTL
	}
	userUUID, err := storekit.ProfileUUID(userID)
	if err != nil {
		return FriendCode{}, err
	}
	for attempt := 0; attempt < 8; attempt++ {
		code, err := randomFriendCode()
		if err != nil {
			return FriendCode{}, err
		}
		expiresAt := time.Now().Add(ttl)
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return FriendCode{}, err
		}
		q := db.New(tx)
		_ = q.RevokeFriendCodes(ctx, userUUID)
		err = q.InsertFriendCode(ctx, db.InsertFriendCodeParams{Code: code, UserID: userUUID, ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true}})
		if err == nil {
			err = tx.Commit(ctx)
			return FriendCode{Code: code, ExpiresAt: expiresAt}, err
		}
		_ = tx.Rollback(ctx)
	}
	return FriendCode{}, errors.New("could not allocate friend code")
}

func (s *PGStore) ResolveFriendCode(ctx context.Context, userID, code string) (CompactPlayer, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	userUUID, err := storekit.ProfileUUID(userID)
	if err != nil {
		return CompactPlayer{}, ErrNotFound
	}
	seasonID, err := storekit.ActiveSeasonID(ctx, s.pool)
	if err != nil {
		return CompactPlayer{}, ErrNotFound
	}
	row, err := db.New(s.pool).ResolveFriendCode(ctx, db.ResolveFriendCodeParams{SeasonID: seasonID, Code: code, SelfUserID: userUUID})
	if err != nil {
		return CompactPlayer{}, ErrNotFound
	}
	p := CompactPlayer{UserID: storekit.UUIDVal(row.UserID), DisplayName: row.DisplayName, AvatarURL: row.AvatarUrl, MMR: int(row.Mmr)}
	if row.LastSeenAt.Valid {
		value := row.LastSeenAt.Time
		p.LastSeenAt = &value
	}
	p.Relationship, p.RequestID, _ = s.Relationship(ctx, userID, p.UserID)
	return p, nil
}

func randomFriendCode() (string, error) {
	buf := make([]byte, 6)
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = friendCodeAlphabet[int(random[i])%len(friendCodeAlphabet)]
	}
	return string(buf), nil
}

const friendCodeAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

func (s *PGStore) CreatePartyInvitation(ctx context.Context, partyID, inviterID, recipientID string, ttl time.Duration) (PartyInvitation, error) {
	if ttl <= 0 {
		ttl = DefaultPartyInviteTTL
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PartyInvitation{}, err
	}
	defer tx.Rollback(ctx)
	partyUUID, err := storekit.ProfileUUID(partyID)
	if err != nil {
		return PartyInvitation{}, ErrBlocked
	}
	inviterUUID, err := storekit.ProfileUUID(inviterID)
	if err != nil {
		return PartyInvitation{}, ErrBlocked
	}
	recipientUUID, err := storekit.ProfileUUID(recipientID)
	if err != nil {
		return PartyInvitation{}, ErrBlocked
	}
	q := db.New(tx)
	eligibility, err := q.PartyInvitationEligibility(ctx, db.PartyInvitationEligibilityParams{PartyID: partyUUID, InviterUserID: inviterUUID, RecipientUserID: recipientUUID})
	if err != nil {
		return PartyInvitation{}, ErrBlocked
	}
	invitation := PartyInvitation{PartyID: partyID, InviteCode: eligibility.InviteCode, Mode: string(eligibility.Mode), MemberCount: int(eligibility.MemberCount)}
	existing, err := q.GetPendingPartyInvitation(ctx, db.GetPendingPartyInvitationParams{PartyID: partyUUID, RecipientUserID: recipientUUID})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return PartyInvitation{}, err
	}
	if err == nil && time.Since(existing.CreatedAt.Time) < PartyInviteResendAfter {
		invitation.ID = storekit.UUIDVal(existing.ID)
		invitation.CreatedAt = existing.CreatedAt.Time
		invitation.ExpiresAt = existing.ExpiresAt.Time
		if err := tx.Commit(ctx); err != nil {
			return PartyInvitation{}, err
		}
		return invitation, nil
	}
	id := entityid.New()
	now := time.Now()
	expiresAt := now.Add(ttl)
	invitationUUID, err := storekit.ProfileUUID(id)
	if err != nil {
		return PartyInvitation{}, err
	}
	row, err := q.UpsertPartyInvitation(ctx, db.UpsertPartyInvitationParams{
		ID: invitationUUID, PartyID: partyUUID, InviterUserID: inviterUUID, RecipientUserID: recipientUUID,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true}, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return PartyInvitation{}, err
	}
	id, expiresAt = storekit.UUIDVal(row.ID), row.ExpiresAt.Time
	var notificationID int64
	_ = storekit.UpsertUserNotificationTx(ctx, tx, recipientID, "party_invitation_received", "party_invitation:"+id,
		map[string]any{"invitationId": id, "actorUserId": inviterID}, &notificationID)
	if err := tx.Commit(ctx); err != nil {
		return PartyInvitation{}, err
	}
	invitation.ID = id
	invitation.CreatedAt = row.CreatedAt.Time
	invitation.ExpiresAt = expiresAt
	return invitation, nil
}

func (s *PGStore) ListPartyInviteStatus(ctx context.Context, inviterID, partyID string) (map[string]CompactPartyInvite, error) {
	out := map[string]CompactPartyInvite{}
	partyUUID, err := storekit.ProfileUUID(partyID)
	if err != nil {
		return out, nil
	}
	inviterUUID, err := storekit.ProfileUUID(inviterID)
	if err != nil {
		return out, nil
	}
	rows, err := db.New(s.pool).ListOutgoingPartyInvitations(ctx, db.ListOutgoingPartyInvitationsParams{
		InviterUserID: inviterUUID, PartyID: partyUUID,
	})
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[storekit.UUIDVal(row.RecipientUserID)] = CompactPartyInvite{ID: storekit.UUIDVal(row.ID), CreatedAt: row.CreatedAt.Time, ExpiresAt: row.ExpiresAt.Time}
	}
	return out, nil
}

func (s *PGStore) ListPartyInvitations(ctx context.Context, userID string, limit int) ([]PartyInvitation, error) {
	limit = BoundedLimit(limit, 10, 50)
	userUUID, err := storekit.ProfileUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := db.New(s.pool).ListPartyInvitations(ctx, db.ListPartyInvitationsParams{RecipientUserID: userUUID, Limit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]PartyInvitation, 0, len(rows))
	for _, row := range rows {
		item := PartyInvitation{ID: storekit.UUIDVal(row.InvitationID), PartyID: storekit.UUIDVal(row.PartyID), InviteCode: row.InviteCode, Mode: string(row.Mode), MemberCount: int(row.MemberCount), CreatedAt: row.CreatedAt.Time, ExpiresAt: row.ExpiresAt.Time}
		item.Inviter = CompactPlayer{UserID: storekit.UUIDVal(row.InviterID), DisplayName: row.DisplayName, AvatarURL: row.AvatarUrl}
		out = append(out, item)
	}
	return out, nil
}

func (s *PGStore) RespondPartyInvitation(ctx context.Context, userID, invitationID, response string) (PartyInvitation, error) {
	status := "declined"
	if response == "accept" {
		status = "accepted"
	}
	userUUID, err := storekit.ProfileUUID(userID)
	if err != nil {
		return PartyInvitation{}, ErrNotFound
	}
	invitationUUID, err := storekit.ProfileUUID(invitationID)
	if err != nil {
		return PartyInvitation{}, ErrNotFound
	}
	q := db.New(s.pool)
	row, err := q.RespondPartyInvitation(ctx, db.RespondPartyInvitationParams{ID: invitationUUID, RecipientUserID: userUUID, Status: db.GdSocialRequestStatus(status)})
	if err != nil {
		return PartyInvitation{}, ErrNotFound
	}
	_ = q.MarkPartyInvitationNotificationRead(ctx, db.MarkPartyInvitationNotificationReadParams{UserID: userUUID, InvitationID: storekit.IngestText(invitationID)})
	return PartyInvitation{ID: storekit.UUIDVal(row.InvitationID), PartyID: storekit.UUIDVal(row.PartyID), InviteCode: row.InviteCode, Mode: string(row.Mode), ExpiresAt: row.ExpiresAt.Time}, nil
}

// TouchLastSeen records the viewer's last-seen timestamp for presence.
func (s *PGStore) TouchLastSeen(ctx context.Context, userID string, seenAt time.Time) error {
	id, err := storekit.ProfileUUID(userID)
	if err != nil {
		return err
	}
	return s.q().TouchLastSeen(ctx, db.TouchLastSeenParams{ID: id, LastSeenAt: pgtype.Timestamptz{Time: seenAt, Valid: true}})
}
