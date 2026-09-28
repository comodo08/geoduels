package staff

import (
	"context"
	"io"

	"geoduels/internal/content"
	"geoduels/internal/maps"
	"geoduels/internal/seasons"
	"geoduels/pkg/contracts"
	"geoduels/pkg/maintenance"
	pkgstaff "geoduels/pkg/staff"
)

// ---- Curation ----
//
// Curation operations share the cycle transaction with any due award.

func (s *Service) NominateMap(ctx context.Context, actor Actor, mapID string) error {
	if err := s.require(actor, pkgstaff.CapCurate); err != nil {
		return err
	}
	if s.store == nil {
		return ErrUnavailable
	}
	return s.withCurationCycle(ctx, func(store Store, cycle CurationCycle) error {
		return store.NominateMap(ctx, actor.ID, mapID, cycle.StartsAt)
	})
}

func (s *Service) LikeNomination(ctx context.Context, actor Actor, id int64, liked bool) error {
	if err := s.require(actor, pkgstaff.CapCurate); err != nil {
		return err
	}
	if s.store == nil {
		return ErrUnavailable
	}
	return s.withCurationCycle(ctx, func(store Store, cycle CurationCycle) error {
		return store.SetNominationLike(ctx, actor.ID, id, liked, cycle.StartsAt)
	})
}

func (s *Service) ListNominations(ctx context.Context, actor Actor, page int) (CurationPage, error) {
	if err := s.require(actor, pkgstaff.CapCurate); err != nil {
		return CurationPage{}, err
	}
	if s.store == nil {
		return CurationPage{}, ErrUnavailable
	}
	if page < 1 {
		page = 1
	}
	if page > 100000 {
		page = 100000
	}
	var result CurationPage
	err := s.withCurationCycle(ctx, func(store Store, cycle CurationCycle) error {
		var err error
		result, err = store.ListNominations(ctx, actor.ID, page, cycle)
		return err
	})
	if err != nil {
		return CurationPage{}, err
	}
	return result, nil
}

// RunCurationSweep closes any due weekly cycle. Used by the background loop.
func (s *Service) RunCurationSweep(ctx context.Context) error {
	if s.store == nil {
		return nil
	}
	return s.withCurationCycle(ctx, func(Store, CurationCycle) error { return nil })
}

// ---- Operations & configuration ----

func (s *Service) GetModerationSettings(ctx context.Context, actor Actor) (content.ModerationSettings, error) {
	if err := s.require(actor, pkgstaff.CapManageConfig); err != nil {
		return content.ModerationSettings{}, err
	}
	return s.store.GetModerationSettings()
}

func (s *Service) SetModerationSettings(ctx context.Context, actor Actor, settings content.ModerationSettings) (content.ModerationSettings, error) {
	if err := s.require(actor, pkgstaff.CapManageConfig); err != nil {
		return content.ModerationSettings{}, err
	}
	if err := s.store.SetModerationSettings(settings); err != nil {
		return content.ModerationSettings{}, err
	}
	return settings, nil
}

func (s *Service) GetDiscordSettings(ctx context.Context, actor Actor) (content.DiscordIntegrationSettings, error) {
	if err := s.require(actor, pkgstaff.CapManageConfig); err != nil {
		return content.DiscordIntegrationSettings{}, err
	}
	return s.store.GetDiscordIntegrationSettings()
}

func (s *Service) SetDiscordSettings(ctx context.Context, actor Actor, settings content.DiscordIntegrationSettings) (content.DiscordIntegrationSettings, error) {
	if err := s.require(actor, pkgstaff.CapManageConfig); err != nil {
		return content.DiscordIntegrationSettings{}, err
	}
	settings.ManagedRoleIDs = nil
	if err := s.store.SetDiscordIntegrationSettings(settings); err != nil {
		return content.DiscordIntegrationSettings{}, err
	}
	return s.store.GetDiscordIntegrationSettings()
}

func (s *Service) GetSeasonSettings(ctx context.Context, actor Actor) (seasons.RankedSeasonSettings, error) {
	if err := s.require(actor, pkgstaff.CapManageConfig); err != nil {
		return seasons.RankedSeasonSettings{}, err
	}
	return s.store.GetRankedSeasonSettings()
}

func (s *Service) SetSeasonResetRule(ctx context.Context, actor Actor, monthlyResetDay int) (seasons.RankedSeasonSettings, error) {
	if err := s.require(actor, pkgstaff.CapManageConfig); err != nil {
		return seasons.RankedSeasonSettings{}, err
	}
	return s.store.SetRankedSeasonResetRule(monthlyResetDay)
}

func (s *Service) SetMapCreatorTier(ctx context.Context, actor Actor, userID string, tier *int) (contracts.MapUploadQuota, error) {
	if err := s.require(actor, pkgstaff.CapManageMaps); err != nil {
		return contracts.MapUploadQuota{}, err
	}
	return s.store.SetMapCreatorTierOverride(userID, tier)
}

func (s *Service) GetMaintenance(ctx context.Context, actor Actor) (maintenance.Status, error) {
	if err := s.require(actor, pkgstaff.CapManageConfig); err != nil {
		return maintenance.Status{}, err
	}
	return s.store.GetMaintenance(ctx)
}

func (s *Service) SetMaintenance(ctx context.Context, actor Actor, status maintenance.Status) (maintenance.Status, error) {
	if err := s.require(actor, pkgstaff.CapManageConfig); err != nil {
		return maintenance.Status{}, err
	}
	status = status.Normalized()
	if err := s.store.SetMaintenance(ctx, status); err != nil {
		return maintenance.Status{}, err
	}
	return status, nil
}

func (s *Service) ClearMaintenance(ctx context.Context, actor Actor) error {
	if err := s.require(actor, pkgstaff.CapManageConfig); err != nil {
		return err
	}
	return s.store.ClearMaintenance(ctx)
}

// ---- Content ----

func (s *Service) GetLobbyChangelog(ctx context.Context, defaultContent content.LobbyChangelogContent) (content.LobbyChangelogContent, error) {
	return s.store.GetLobbyChangelog(defaultContent)
}

func (s *Service) ListChangelogPosts(ctx context.Context, includeUnpublished bool) ([]content.ChangelogPost, error) {
	return s.store.ListChangelogPosts(includeUnpublished)
}

func (s *Service) GetChangelogPost(ctx context.Context, slug string, publishedOnly bool) (content.ChangelogPost, bool, error) {
	return s.store.GetChangelogPostBySlug(slug, publishedOnly)
}

func (s *Service) CreateChangelogPost(ctx context.Context, actor Actor, input content.ChangelogPostInput) (content.ChangelogPost, error) {
	if err := s.require(actor, pkgstaff.CapManageContent); err != nil {
		return content.ChangelogPost{}, err
	}
	return s.store.CreateChangelogPost(input)
}

func (s *Service) UpdateChangelogPost(ctx context.Context, actor Actor, id int64, input content.ChangelogPostInput) (content.ChangelogPost, bool, error) {
	if err := s.require(actor, pkgstaff.CapManageContent); err != nil {
		return content.ChangelogPost{}, false, err
	}
	return s.store.UpdateChangelogPost(id, input)
}

// ---- Maps administration ----

func (s *Service) ImportOfficialMap(ctx context.Context, actor Actor, input maps.OfficialMapImportInput, source io.Reader) (contracts.CustomMap, error) {
	if err := s.require(actor, pkgstaff.CapManageMaps); err != nil {
		return contracts.CustomMap{}, err
	}
	return s.store.ImportOfficialMap(actor.ID, input, source)
}

func (s *Service) ReplaceMapLocations(ctx context.Context, actor Actor, mapKey, displayName string, dataset []byte) (contracts.MapImportSummary, error) {
	if err := s.require(actor, pkgstaff.CapManageMaps); err != nil {
		return contracts.MapImportSummary{}, err
	}
	return s.store.ReplaceMapLocations(mapKey, displayName, dataset)
}
