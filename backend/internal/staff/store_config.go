package staff

import (
	"io"

	"geoduels/internal/content"
	"geoduels/internal/maps"
	"geoduels/internal/seasons"
	"geoduels/pkg/contracts"
)

func (a *PGStore) GetModerationSettings() (content.ModerationSettings, error) {
	return a.content.GetModerationSettings()
}
func (a *PGStore) SetModerationSettings(settings content.ModerationSettings) error {
	return a.content.SetModerationSettings(settings)
}
func (a *PGStore) GetDiscordIntegrationSettings() (content.DiscordIntegrationSettings, error) {
	return a.content.GetDiscordIntegrationSettings()
}
func (a *PGStore) SetDiscordIntegrationSettings(settings content.DiscordIntegrationSettings) error {
	return a.content.SetDiscordIntegrationSettings(settings)
}

func (a *PGStore) GetLobbyChangelog(defaultContent content.LobbyChangelogContent) (content.LobbyChangelogContent, error) {
	return a.content.GetLobbyChangelog(defaultContent)
}
func (a *PGStore) ListChangelogPosts(includeUnpublished bool) ([]content.ChangelogPost, error) {
	return a.content.ListChangelogPosts(includeUnpublished)
}
func (a *PGStore) GetChangelogPostBySlug(slug string, publishedOnly bool) (content.ChangelogPost, bool, error) {
	return a.content.GetChangelogPostBySlug(slug, publishedOnly)
}
func (a *PGStore) CreateChangelogPost(input content.ChangelogPostInput) (content.ChangelogPost, error) {
	return a.content.CreateChangelogPost(input)
}
func (a *PGStore) UpdateChangelogPost(id int64, input content.ChangelogPostInput) (content.ChangelogPost, bool, error) {
	return a.content.UpdateChangelogPost(id, input)
}

func (a *PGStore) GetRankedSeasonSettings() (seasons.RankedSeasonSettings, error) {
	return a.seasons.GetRankedSeasonSettings()
}
func (a *PGStore) SetRankedSeasonResetRule(monthlyResetDay int) (seasons.RankedSeasonSettings, error) {
	return a.seasons.SetRankedSeasonResetRule(monthlyResetDay)
}

func (a *PGStore) SetMapCreatorTierOverride(userID string, tier *int) (contracts.MapUploadQuota, error) {
	return a.maps.SetMapCreatorTierOverride(userID, tier)
}
func (a *PGStore) ImportOfficialMap(adminUserID string, input maps.OfficialMapImportInput, source io.Reader) (contracts.CustomMap, error) {
	return a.maps.ImportOfficialMap(adminUserID, input, source)
}
func (a *PGStore) ReplaceMapLocations(mapKey, displayName string, dataset []byte) (contracts.MapImportSummary, error) {
	return a.maps.ReplaceMapLocations(mapKey, displayName, dataset)
}
