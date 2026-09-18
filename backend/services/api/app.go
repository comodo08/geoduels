package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"geoduels/internal/accounts"
	"geoduels/internal/admin"
	"geoduels/internal/authsession"
	"geoduels/internal/badges"
	"geoduels/internal/chat"
	"geoduels/internal/content"
	"geoduels/internal/leaderboard"
	"geoduels/internal/maps"
	"geoduels/internal/matches"
	"geoduels/internal/moderation"
	"geoduels/internal/notifications"
	"geoduels/internal/parties"
	preferencesdomain "geoduels/internal/preferences"
	"geoduels/internal/profiles"
	"geoduels/internal/seasons"
	socialdomain "geoduels/internal/social"
	"geoduels/internal/storage"
	"geoduels/pkg/auth"
	"geoduels/pkg/contracts"
	"geoduels/pkg/coordinator"
	"geoduels/pkg/observability"
	"geoduels/pkg/persistence"
)

type api struct {
	matchCoordinator        string
	db                      *persistence.DB
	accounts                accounts.Store
	sessions                authsession.Store
	profiles                profiles.Store
	badges                  badges.Store
	matchStore              matches.Store
	moderation              moderation.Store
	admin                   admin.Store
	content                 content.Store
	seasons                 seasons.Store
	gameplayMaps            maps.Store
	runtimeStore            matches.Store
	chatStore               chat.Store
	parties                 parties.Store
	storage                 storage.Store
	social                  *socialdomain.Service
	maps                    *maps.Service
	mapsStore               *maps.PGStore
	preferences             *preferencesdomain.Service
	leaderboardService      *leaderboard.Service
	notificationService     *notifications.Service
	authSessionService      *authsession.Service
	coord                   *coordinator.Store
	redis                   *redis.Client
	httpClient              *http.Client
	googleVerifier          *auth.GoogleVerifier
	googleClientID          string
	googleSecret            string
	discordClientID         string
	discordSecret           string
	stripeMode              string
	stripeTestPaymentLink   string
	stripeLivePaymentLink   string
	stripeLegacyPaymentURL  string
	stripeTestWebhook       string
	stripeLiveWebhook       string
	stripeLegacyWebhook     string
	appAuthSecret           []byte
	ticketAuth              []byte
	internalSecret          string
	accessTokenTTL          time.Duration
	refreshTokenTTL         time.Duration
	refreshCookieName       string
	refreshCookieDomain     string
	refreshCookieSameSite   http.SameSite
	guestSignupIPLimit      int
	guestSignupIPWindow     time.Duration
	guestSignupDailyLimit   int
	guestSignupDailyWindow  time.Duration
	guestAccountTTL         time.Duration
	guestCleanupInterval    time.Duration
	guestCleanupBatchSize   int
	storageCleanupInterval  time.Duration
	storageCleanupBatchSize int
	staleMatchGrace         time.Duration
	turnstileSecret         string
	turnstileVerifyURL      string
	turnstileHostname       string
	guestTurnstileRequired  bool
	trustedProxyCIDRs       []*net.IPNet
	adminBootstrapEmails    map[string]struct{}
	metrics                 *observability.APIMetrics
	globalStatus            *globalStatusHub
	live                    *liveHub
	lastSeen                lastSeenWriter
	draining                atomic.Bool
}

func newAPI() (*api, error) {
	store, err := persistence.NewFromEnv()
	if err != nil {
		return nil, err
	}
	rdb, _, err := redisFromEnv()
	if err != nil {
		store.Close()
		return nil, err
	}
	googleClientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	googleSecret := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET"))
	discordClientID := strings.TrimSpace(os.Getenv("DISCORD_CLIENT_ID"))
	discordSecret := strings.TrimSpace(os.Getenv("DISCORD_CLIENT_SECRET"))
	stripeMode := strings.TrimSpace(strings.ToLower(os.Getenv("STRIPE_MODE")))
	stripeTestPaymentLink := strings.TrimSpace(os.Getenv("STRIPE_TEST_PAYMENT_LINK_URL"))
	stripeLivePaymentLink := strings.TrimSpace(os.Getenv("STRIPE_LIVE_PAYMENT_LINK_URL"))
	stripeLegacyPaymentURL := strings.TrimSpace(os.Getenv("STRIPE_PAYMENT_LINK_URL"))
	stripeTestWebhook := strings.TrimSpace(os.Getenv("STRIPE_TEST_WEBHOOK_SECRET"))
	stripeLiveWebhook := strings.TrimSpace(os.Getenv("STRIPE_LIVE_WEBHOOK_SECRET"))
	stripeLegacyWebhook := strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET"))
	var googleVerifier *auth.GoogleVerifier
	if googleClientID != "" && googleSecret != "" {
		googleVerifier, err = auth.NewGoogleVerifier(context.Background(), googleClientID, getenv("GOOGLE_ISSUER", ""))
		if err != nil {
			store.Close()
			return nil, err
		}
	}
	appAuthSecret, err := requiredSecret("APP_AUTH_SECRET", 32)
	if err != nil {
		store.Close()
		return nil, err
	}
	ticketAuth, err := requiredSecret("GAMEPLAY_TICKET_SECRET", 32)
	if err != nil {
		store.Close()
		return nil, err
	}
	internalSecret := strings.TrimSpace(os.Getenv("COORDINATOR_INTERNAL_SECRET"))
	if internalSecret == "" {
		store.Close()
		return nil, errors.New("COORDINATOR_INTERNAL_SECRET is required")
	}
	trustedProxyCIDRs, err := parseCIDRs(os.Getenv("TRUSTED_PROXY_CIDRS"))
	if err != nil {
		store.Close()
		return nil, err
	}
	guestTurnstileRequired := getenvBool("TURNSTILE_GUEST_REQUIRED", false)
	turnstileSecret := strings.TrimSpace(os.Getenv("TURNSTILE_SECRET_KEY"))
	if guestTurnstileRequired && turnstileSecret == "" {
		store.Close()
		return nil, errors.New("TURNSTILE_SECRET_KEY is required when TURNSTILE_GUEST_REQUIRED=true")
	}
	singleplayerTTL := getenvDuration("SINGLEPLAYER_SESSION_TTL", 24*time.Hour)
	pool := store.Pool()
	mapsStore := maps.NewPGStore(pool)
	matchStore := matches.NewPGStore(pool)
	moderationStore := moderation.NewPGStore(pool)
	matchStore.CheatBans = moderationStore
	partyStore := parties.NewPGStore(pool, mapsStore)
	if err := matchStore.ExpireStaleRuntimeMatches(context.Background(), string(contracts.ModeSingleplayer), singleplayerTTL); err != nil {
		store.Close()
		return nil, err
	}
	if err := partyStore.ExpireOpenParties(); err != nil {
		store.Close()
		return nil, err
	}
	socialStore := socialdomain.NewPGStore(pool)
	instance := &api{
		matchCoordinator:        getenv("MATCH_COORDINATOR_URL", getenv("QUEUE_COORDINATOR_URL", "http://localhost:8090")),
		db:                      store,
		accounts:                accounts.NewPGStore(pool),
		sessions:                authsession.NewPGStore(pool),
		profiles:                profiles.NewPGStore(pool),
		badges:                  badges.NewPGStore(pool),
		matchStore:              matchStore,
		moderation:              moderationStore,
		admin:                   admin.NewPGStore(pool),
		content:                 content.NewPGStore(pool),
		seasons:                 seasons.NewPGStore(pool),
		gameplayMaps:            mapsStore,
		runtimeStore:            matchStore,
		chatStore:               chat.NewPGStore(pool),
		parties:                 partyStore,
		storage:                 storage.NewPGStore(pool),
		social:                  socialdomain.NewService(socialStore),
		maps:                    maps.NewService(mapsStore),
		mapsStore:               mapsStore,
		lastSeen:                socialStore,
		preferences:             preferencesdomain.NewService(preferencesdomain.NewPGStore(pool)),
		leaderboardService:      leaderboard.NewService(leaderboard.NewPGStore(pool)),
		notificationService:     notifications.NewService(notifications.NewPGStore(pool)),
		authSessionService:      authsession.NewService(authsession.NewPGStore(pool)),
		coord:                   coordinator.NewStore(rdb, getenvDuration("GAMEPLAY_NODE_TTL", 10*time.Second), 2*time.Hour, singleplayerTTL, 5*time.Second),
		redis:                   rdb,
		httpClient:              &http.Client{Timeout: 3 * time.Second},
		googleVerifier:          googleVerifier,
		googleClientID:          googleClientID,
		googleSecret:            googleSecret,
		discordClientID:         discordClientID,
		discordSecret:           discordSecret,
		stripeMode:              stripeMode,
		stripeTestPaymentLink:   stripeTestPaymentLink,
		stripeLivePaymentLink:   stripeLivePaymentLink,
		stripeLegacyPaymentURL:  stripeLegacyPaymentURL,
		stripeTestWebhook:       stripeTestWebhook,
		stripeLiveWebhook:       stripeLiveWebhook,
		stripeLegacyWebhook:     stripeLegacyWebhook,
		appAuthSecret:           appAuthSecret,
		ticketAuth:              ticketAuth,
		internalSecret:          internalSecret,
		accessTokenTTL:          getenvDuration("APP_ACCESS_TOKEN_TTL", 15*time.Minute),
		refreshTokenTTL:         getenvDuration("APP_REFRESH_TOKEN_TTL", 30*24*time.Hour),
		refreshCookieName:       getenv("APP_REFRESH_COOKIE_NAME", "geoduels_refresh"),
		refreshCookieDomain:     strings.TrimSpace(os.Getenv("APP_REFRESH_COOKIE_DOMAIN")),
		refreshCookieSameSite:   getenvSameSite("APP_REFRESH_COOKIE_SAMESITE", http.SameSiteLaxMode),
		guestSignupIPLimit:      getenvInt("GUEST_SIGNUP_IP_LIMIT", 5),
		guestSignupIPWindow:     getenvDuration("GUEST_SIGNUP_IP_WINDOW", 10*time.Minute),
		guestSignupDailyLimit:   getenvInt("GUEST_SIGNUP_IP_DAILY_LIMIT", 10),
		guestSignupDailyWindow:  getenvDuration("GUEST_SIGNUP_IP_DAILY_WINDOW", 24*time.Hour),
		guestAccountTTL:         getenvDuration("GUEST_ACCOUNT_TTL", 24*time.Hour),
		guestCleanupInterval:    getenvDuration("GUEST_ACCOUNT_CLEANUP_INTERVAL", time.Hour),
		guestCleanupBatchSize:   getenvInt("GUEST_ACCOUNT_CLEANUP_BATCH_SIZE", 1000),
		storageCleanupInterval:  getenvDuration("STORAGE_CLEANUP_INTERVAL", time.Minute),
		storageCleanupBatchSize: getenvInt("STORAGE_CLEANUP_BATCH_SIZE", 1000),
		staleMatchGrace:         getenvDuration("MATCH_SESSION_STALE_GRACE", 5*time.Minute),
		turnstileSecret:         turnstileSecret,
		turnstileVerifyURL:      getenv("TURNSTILE_VERIFY_URL", turnstileSiteverifyURL),
		turnstileHostname:       strings.TrimSpace(os.Getenv("TURNSTILE_EXPECTED_HOSTNAME")),
		guestTurnstileRequired:  guestTurnstileRequired,
		trustedProxyCIDRs:       trustedProxyCIDRs,
		adminBootstrapEmails:    parseEmailAllowlist(os.Getenv("ADMIN_BOOTSTRAP_EMAILS")),
		metrics:                 observability.NewAPIMetrics(),
	}
	instance.globalStatus = newGlobalStatusHub(instance)
	instance.globalStatus.start()
	instance.live = newLiveHub(instance)
	instance.live.start()
	return instance, nil
}

func routes(a *api) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}
		code := http.StatusInternalServerError
		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
		}
		// gorilla/mux wrote empty bodies for unrouted paths and methods.
		if code == http.StatusNotFound || code == http.StatusMethodNotAllowed {
			_ = c.NoContent(code)
			return
		}
		e.DefaultHTTPErrorHandler(err, c)
	}
	e.Use(corsMiddleware)
	if a.metrics != nil {
		e.Use(a.metrics.EchoMiddleware)
	}

	e.GET("/health", a.healthReady)
	e.GET("/health/live", a.healthLive)
	e.GET("/health/ready", a.healthReady)
	e.POST("/v1/auth/guest", a.guestLogin)
	e.POST("/v1/auth/google/start", a.googleOAuthStart)
	e.GET("/v1/auth/google/callback", a.googleOAuthCallback)
	e.POST("/v1/auth/discord/start", a.discordOAuthStart)
	e.GET("/v1/auth/discord/callback", a.discordOAuthCallback)
	e.GET("/v1/bootstrap", a.bootstrap)
	e.POST("/v1/auth/refresh", a.refresh)
	e.POST("/v1/auth/logout", a.logout)
	e.POST("/v1/auth/logout-all", a.logoutAll)
	e.GET("/v1/status", a.publicGlobalStatus)
	e.POST("/v1/admin/bootstrap", a.adminBootstrap)
	e.PATCH("/v1/me/badge", a.updateSelectedBadge, a.active)
	e.PUT("/v1/me/nickname", a.updateNickname, a.active)
	e.PATCH("/v1/me/nickname", a.updateNickname, a.active)
	e.PATCH("/v1/me/preferences", a.updateUserPreferences, a.active)
	e.DELETE("/v1/me", a.deleteAccount)
	e.DELETE("/v1/me/auth-providers/:provider", a.unlinkAuthProvider)
	e.GET("/v1/me/live", a.userLive)
	e.GET("/v1/me/notifications", a.userNotifications)
	e.POST("/v1/me/notifications/read-all", a.markAllUserNotificationsRead)
	e.POST("/v1/me/notifications/:id/read", a.markUserNotificationRead)
	e.GET("/v1/me/social-settings", a.socialSettings)
	e.PATCH("/v1/me/social-settings", a.socialSettings, a.active)
	e.GET("/v1/me/friends-page", a.friendsPage)
	e.POST("/v1/me/friend-code", a.createFriendCode, a.active)
	e.GET("/v1/me/party-invitations", a.partyInvitations)
	e.POST("/v1/friend-requests", a.sendFriendRequest, a.active)
	e.POST("/v1/friend-requests/:id/:action", a.respondFriendRequest, a.active)
	e.DELETE("/v1/friends/:userId", a.removeFriend, a.active)
	e.POST("/v1/blocks/:userId", a.userBlock, a.active)
	e.DELETE("/v1/blocks/:userId", a.userBlock, a.active)
	e.GET("/v1/friend-codes/:code", a.resolveFriendCode)
	e.POST("/v1/friend-codes/:code/request", a.sendFriendCodeRequest, a.active)
	e.POST("/v1/parties/:id/invitations", a.partyInvitations, a.active)
	e.POST("/v1/party-invitations/:id/:action", a.respondPartyInvitation, a.active)
	e.POST("/v1/party-invitations", a.createPartyAndInvite, a.active)
	e.POST("/v1/support/donate", a.createSupportDonation, a.active)
	e.POST("/v1/integrations/stripe/webhook", a.stripeWebhook)
	e.GET("/v1/content/lobby-changelog", a.publicLobbyChangelog)
	e.GET("/v1/content/changelog", a.publicChangelogPosts)
	e.GET("/v1/content/changelog/:slug", a.publicChangelogPost)

	e.GET("/v1/leaderboard", a.leaderboard)
	e.GET("/v1/players/:nickname", a.publicPlayerProfile)
	e.GET("/v1/players/:nickname/matches", a.publicPlayerMatches)
	e.GET("/v1/players/:nickname/relationship", a.playerRelationship)
	e.GET("/v1/player-search", a.socialPlayerSearch)
	e.GET("/v1/matches/:id", a.match)
	e.GET("/v1/matches/:id/bootstrap", a.matchBootstrap)
	e.GET("/v1/matches/:id/route", a.matchRoute)
	e.GET("/v1/matches/:id/session", a.matchSession, a.active)
	e.POST("/v1/matches/:id/reports", a.createMatchReport, a.active)
	e.POST("/v1/sessions", a.startSession, a.active)
	e.POST("/v1/singleplayer/session", a.startSingleplayerSession, a.active)
	e.GET("/v1/maps", a.listMaps)
	e.POST("/v1/maps", a.createMap, a.active)
	e.GET("/v1/maps/quota", a.mapUploadQuota)
	e.GET("/v1/maps/:id", a.getMap)
	e.PATCH("/v1/maps/:id", a.updateMap, a.active)
	e.DELETE("/v1/maps/:id", a.archiveMap, a.active)
	e.POST("/v1/maps/:id/publish", a.publishMap, a.active)
	e.POST("/v1/maps/:id/official", a.setMapOfficial)
	e.DELETE("/v1/maps/:id/official", a.unsetMapOfficial)
	e.POST("/v1/maps/:id/roles/:role", a.setGameplayMapRole)
	e.POST("/v1/maps/:id/favorite", a.favoriteMap, a.active)
	e.DELETE("/v1/maps/:id/favorite", a.unfavoriteMap, a.active)
	e.GET("/v1/maps/:id/comments", a.listMapComments)
	e.POST("/v1/maps/:id/comments", a.createMapComment, a.active)
	e.DELETE("/v1/maps/:id/comments/:commentId", a.deleteMapComment, a.active)
	e.POST("/v1/maps/:id/comments/:commentId/like", a.likeMapComment, a.active)
	e.DELETE("/v1/maps/:id/comments/:commentId/like", a.unlikeMapComment, a.active)
	e.PUT("/v1/maps/:id/locations", a.replaceMapLocations, a.active)
	e.GET("/v1/admin/players", a.adminPlayers)
	e.GET("/v1/admin/players/:id", a.adminPlayerDetail)
	e.GET("/v1/admin/players/:id/matches", a.adminPlayerMatches)
	e.POST("/v1/admin/players/:id/ban", a.adminBanPlayer)
	e.POST("/v1/admin/players/:id/unban", a.adminUnbanPlayer)
	e.DELETE("/v1/admin/players/:id/report-mute", a.adminClearReporterMute)
	e.GET("/v1/admin/moderation/community-pardon", a.adminCommunityPardonPreview)
	e.POST("/v1/admin/moderation/community-pardon", a.adminCommunityPardon)
	e.POST("/v1/admin/players/:id/moderator", a.adminPromoteModerator)
	e.DELETE("/v1/admin/players/:id/moderator", a.adminDemoteModerator)
	e.PUT("/v1/admin/players/:id/map-tier", a.adminSetMapCreatorTier)
	e.GET("/v1/admin/roles", a.adminListRoles)
	e.POST("/v1/admin/roles", a.adminGrantRole)
	e.DELETE("/v1/admin/roles/:id/:role", a.adminRevokeRole)
	e.GET("/v1/admin/badges", a.adminBadgeDefinitions)
	e.POST("/v1/admin/badges/grant", a.adminGrantBadge)
	e.GET("/v1/admin/matches/:id/chat", a.adminMatchChat)
	e.GET("/v1/moderator/subjects/:userId", a.moderatorSubject)
	e.POST("/v1/moderator/subjects/:userId/cheating-ban", a.moderatorSubjectCheatingBan)
	e.POST("/v1/moderator/subjects/:userId/unban", a.moderatorSubjectUnban)
	e.POST("/v1/moderator/subjects/:userId/mutes/:kind", a.moderatorSubjectMute)
	e.DELETE("/v1/moderator/subjects/:userId/mutes/:kind", a.moderatorSubjectUnmute)
	e.GET("/v1/moderator/signals", a.moderatorSignals)
	e.GET("/v1/moderator/log", a.moderatorLog)
	e.GET("/v1/admin/ip-signup-bans", a.adminListSignupIPBans)
	e.POST("/v1/admin/ip-signup-bans", a.adminAddSignupIPBan)
	e.DELETE("/v1/admin/ip-signup-bans/:ip", a.adminRemoveSignupIPBan)
	e.GET("/v1/admin/maintenance", a.adminGetMaintenance)
	e.PUT("/v1/admin/maintenance", a.adminPutMaintenance)
	e.DELETE("/v1/admin/maintenance", a.adminClearMaintenance)
	e.GET("/v1/admin/moderation/settings", a.adminGetModerationSettings)
	e.PUT("/v1/admin/moderation/settings", a.adminPutModerationSettings)
	e.GET("/v1/admin/integrations/discord", a.adminGetDiscordIntegrationSettings)
	e.PUT("/v1/admin/integrations/discord", a.adminPutDiscordIntegrationSettings)
	e.GET("/v1/admin/seasons", a.adminGetRankedSeason)
	e.PUT("/v1/admin/seasons/reset-rule", a.adminPutRankedSeasonResetRule)
	e.GET("/v1/admin/changelog", a.adminGetChangelog)
	e.POST("/v1/admin/changelog", a.adminCreateChangelogPost)
	e.PUT("/v1/admin/changelog/:id", a.adminUpdateChangelogPost)
	e.POST("/v1/admin/maps/official/import", a.adminImportOfficialMap)
	e.POST("/v1/admin/maps/current/upload", a.adminUploadCurrentMap)
	e.POST("/v1/admin/maps/:mapKey/upload", a.adminUploadMap)
	if a.metrics != nil {
		e.GET("/metrics", echo.WrapHandler(observability.Handler(a.metrics.Registry)))
	}
	return e
}

func parseEmailAllowlist(raw string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		email := strings.ToLower(strings.TrimSpace(part))
		if email == "" {
			continue
		}
		out[email] = struct{}{}
	}
	return out
}
