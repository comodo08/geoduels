package parties

import (
	"time"

	"geoduels/pkg/contracts"
)

type Store interface {
	ExpireOpenParties() error
	ListOpenPartyIDs() ([]string, error)
	CloseInactiveOpenParties(lobbyIDs []string, inactiveFor time.Duration) (int64, error)
	CreateParty(ownerUserID string, mode contracts.MatchMode, mapScope string, ttl time.Duration) (contracts.PartySnapshot, error)
	SetPartyMode(lobbyID string, mode contracts.MatchMode) error
	SetPartyConfig(lobbyID string, config contracts.MatchConfig) (contracts.PartySnapshot, error)
	GetPartyByID(lobbyID string) (contracts.PartySnapshot, bool, error)
	GetPartyByInviteCode(inviteCode string) (contracts.PartySnapshot, bool, error)
	GetPartyByMatchID(matchID string) (contracts.PartySnapshot, bool, error)
	JoinParty(lobbyID, userID string) (contracts.PartySnapshot, error)
	LeaveParty(lobbyID, userID string) (contracts.PartySnapshot, error)
	SetPartyMemberTeam(lobbyID, userID, teamID string) (contracts.PartySnapshot, error)
	KickPartyMember(lobbyID, ownerUserID, targetUserID string) (contracts.PartySnapshot, error)
	TransferPartyOwner(lobbyID, ownerUserID, targetUserID string) (contracts.PartySnapshot, error)
	MarkPartyInMatch(lobbyID, matchID string) (contracts.PartySnapshot, error)
	ReopenEndedParties() (int64, error)
}

type Service struct{ Store }

func NewService(store Store) *Service { return &Service{Store: store} }
