package content

type Store interface {
	GetLobbyChangelog(defaultContent LobbyChangelogContent) (LobbyChangelogContent, error)
	SetLobbyChangelog(content LobbyChangelogContent) error
	ListChangelogPosts(includeUnpublished bool) ([]ChangelogPost, error)
	GetChangelogPostBySlug(slug string, publishedOnly bool) (ChangelogPost, bool, error)
	CreateChangelogPost(input ChangelogPostInput) (ChangelogPost, error)
	UpdateChangelogPost(id int64, input ChangelogPostInput) (ChangelogPost, bool, error)
	GetModerationSettings() (ModerationSettings, error)
	SetModerationSettings(settings ModerationSettings) error
	GetDiscordIntegrationSettings() (DiscordIntegrationSettings, error)
	SetDiscordIntegrationSettings(settings DiscordIntegrationSettings) error
}

type Service struct{ Store }

func NewService(store Store) *Service { return &Service{Store: store} }
