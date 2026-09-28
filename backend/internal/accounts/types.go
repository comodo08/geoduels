package accounts

import "geoduels/pkg/staff"

const (
	IdentityProviderGoogle  = "google"
	IdentityProviderDiscord = "discord"

	DiscordSyncActionSync         = "sync"
	DiscordSyncActionCleanupRoles = "cleanup_roles"
)

type Identity struct {
	Roles                 staff.Roles
	Sub                   string
	Email                 string
	GoogleName            string
	ProviderName          string
	AvatarURL             string
	NicknameRequired      bool
	DisplayName           string
	IsGuest               bool
	LinkedProviders       []string
	AuthMigrationRequired bool
	RecoveryAvailable     bool
	IsAdmin               bool
	IsModerator           bool
	IsBanned              bool
	BanReason             string
}

func (i Identity) StaffActor() staff.Actor {
	return staff.Actor{ID: i.Sub, Roles: i.Roles, Banned: i.IsBanned}
}
