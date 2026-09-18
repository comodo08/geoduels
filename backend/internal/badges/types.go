package badges

import "geoduels/pkg/contracts"

const (
	DiscordSyncActionSync         = "sync"
	DiscordSyncActionCleanupRoles = "cleanup_roles"
)

const (
	badgeCodeDiscordMember       = int16(1)
	badgeCodeGeoDuelsTeam        = int16(2)
	badgeCodeDiscordServerMember = int16(3)
	badgeCodeSupporter           = int16(4)
	badgeCodeSpeedrunner         = int16(5)
	badgeCodeElo1000             = int16(6)
	badgeCodeElo1500             = int16(7)
	badgeCodeElo2000             = int16(8)
	badgeCodeLegacyTopFinish     = int16(10)
	badgeCodeTopFinish           = int16(11)
	badgeCodeEventWinner2026     = int16(12)
)

type AdminBadgeDefinition struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Label       string `json:"label"`
	Description string `json:"description"`
	ImageURL    string `json:"imageUrl"`
	Rarity      string `json:"rarity,omitempty"`
	MaxLevel    int    `json:"maxLevel"`
}

type DiscordSyncOutboxItem struct {
	ID            int64
	Action        string
	DiscordUserID string
	Attempts      int
}

type DiscordLinkedUser struct {
	UserID             string
	DiscordUserID      string
	HighestEloBadgeMMR int
}

type PlayerBadge = contracts.PlayerBadge
