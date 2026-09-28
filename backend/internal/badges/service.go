package badges

import (
	"geoduels/pkg/contracts"
)

type Store interface {
	SyncLoginBadges(userID string) error
	AwardDiscordServerMemberByDiscordID(discordUserID string) (bool, error)
	GetDiscordLinkedUser(discordUserID string) (DiscordLinkedUser, bool, error)
	CreateDonationRef(userID string) (string, error)
	AwardSupporterByDonationRef(ref string) (bool, error)
	ListAdminGrantableBadges() []AdminBadgeDefinition
	GrantBadgeToUser(nickname, badgeID, actorUserID string) (contracts.PlayerBadge, bool, error)
}

type Service struct{ Store }

func NewService(store Store) *Service { return &Service{Store: store} }
