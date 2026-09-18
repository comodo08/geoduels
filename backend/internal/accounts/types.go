package accounts

const (
	IdentityProviderGoogle  = "google"
	IdentityProviderDiscord = "discord"

	DiscordSyncActionSync         = "sync"
	DiscordSyncActionCleanupRoles = "cleanup_roles"
)

type Identity struct {
	Sub                   string
	Email                 string
	GoogleName            string
	ProviderName          string
	AvatarURL             string
	NicknameRequired      bool
	DisplayName           string
	AccountType           string
	LinkedProviders       []string
	AuthMigrationRequired bool
	RecoveryAvailable     bool
	IsAdmin               bool
	IsModerator           bool
	IsBanned              bool
	BanReason             string
}
