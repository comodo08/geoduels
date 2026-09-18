package main

import (
	"github.com/jackc/pgx/v5"

	"geoduels/internal/accounts"
	"geoduels/internal/badges"
	"geoduels/internal/content"
	"geoduels/internal/maps"
	"geoduels/internal/moderation"
	socialdomain "geoduels/internal/social"
	"geoduels/pkg/contracts"
)

type (
	Identity                       = accounts.Identity
	RefreshTokenRecord             = contracts.RefreshTokenRecord
	AuthSessionParams              = contracts.AuthSessionParams
	ChangelogPostInput             = content.ChangelogPostInput
	LobbyChangelogContent          = content.LobbyChangelogContent
	ModerationSettings             = content.ModerationSettings
	DiscordIntegrationSettings     = content.DiscordIntegrationSettings
	AdminPlayerSummary             = contracts.AdminPlayerSummary
	UserNotification               = contracts.UserNotification
	MapCreatorAdminRepository      = maps.MapCreatorAdminRepository
	OfficialMapImportInput         = maps.OfficialMapImportInput
	CreatePlayerReportSignalParams = moderation.CreatePlayerReportSignalParams
)

const (
	IdentityProviderGoogle  = accounts.IdentityProviderGoogle
	IdentityProviderDiscord = accounts.IdentityProviderDiscord
)

var (
	ErrNoRows                = pgx.ErrNoRows
	ErrNicknameTaken         = accounts.ErrNicknameTaken
	ErrOAuthEmailConflict    = accounts.ErrOAuthEmailConflict
	ErrBadgeNicknameRequired = badges.ErrBadgeNicknameRequired
	ErrBadgeUnavailable      = badges.ErrBadgeUnavailable
	ErrBadgeUserNotFound     = badges.ErrBadgeUserNotFound
	ErrSocialNotFound        = socialdomain.ErrNotFound
	ErrSocialLimit           = socialdomain.ErrLimit
	ErrSocialBlocked         = socialdomain.ErrBlocked
)
