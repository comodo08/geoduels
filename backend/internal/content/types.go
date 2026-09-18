package content

import "time"

type LobbyChangelogContent struct {
	Eyebrow   string    `json:"eyebrow"`
	Title     string    `json:"title"`
	Markdown  string    `json:"markdown"`
	Slug      string    `json:"slug,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

type ChangelogPost struct {
	ID        int64     `json:"id"`
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	Markdown  string    `json:"markdown"`
	Published bool      `json:"published"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ChangelogPostInput struct {
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Markdown  string `json:"markdown"`
	Published bool   `json:"published"`
}

type ModerationSettings struct {
	DiscordWebhookURL string `json:"discordWebhookUrl"`
}

type DiscordIntegrationSettings struct {
	GuildID                  string   `json:"guildId"`
	JoinsChannelID           string   `json:"joinsChannelId"`
	Elo1000RoleID            string   `json:"elo1000RoleId"`
	Elo1500RoleID            string   `json:"elo1500RoleId"`
	Elo2000RoleID            string   `json:"elo2000RoleId"`
	ManagedRoleIDs           []string `json:"managedRoleIds,omitempty"`
	ReconcileIntervalMinutes int      `json:"reconcileIntervalMinutes"`
}
