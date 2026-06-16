package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"

	"geoduels/pkg/observability"
	"geoduels/pkg/persistence"
)

const (
	workerDrainTimeout    = 5 * time.Second
	workerShutdownTimeout = 10 * time.Second
	discordSyncInterval   = 15 * time.Second
	discordSyncBatch      = 10
)

type rankRoleConfig struct {
	Elo1000 string
	Elo1500 string
	Elo2000 string
}

type worker struct {
	store          persistence.Store
	guildID        string
	joinsChannelID string
	rankRoles      rankRoleConfig
	session        *discordgo.Session
	draining       atomic.Bool
	ready          atomic.Bool
}

func main() {
	w, err := newWorker()
	if err != nil {
		log.Fatal(err)
	}
	defer w.close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := w.openDiscord(); err != nil {
		log.Fatal(err)
	}
	w.startReconciliation(ctx, getenvDuration("DISCORD_RECONCILE_INTERVAL", 15*time.Minute))
	w.startDiscordSyncWorker(ctx)

	r := http.NewServeMux()
	r.HandleFunc("/health/live", w.healthLive)
	r.HandleFunc("/health/ready", w.healthReady)
	r.HandleFunc("/health", w.healthReady)

	addr := getenv("DISCORD_WORKER_ADDR", ":8094")
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	observability.Log("info", "discord worker startup", map[string]any{"addr": addr})
	go handleWorkerShutdown(w, srv, cancel)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func newWorker() (*worker, error) {
	token := strings.TrimSpace(os.Getenv("DISCORD_BOT_TOKEN"))
	if token == "" {
		return nil, errors.New("DISCORD_BOT_TOKEN is required")
	}
	guildID := strings.TrimSpace(os.Getenv("DISCORD_GUILD_ID"))
	if guildID == "" {
		return nil, errors.New("DISCORD_GUILD_ID is required")
	}
	joinsChannelID := strings.TrimSpace(os.Getenv("DISCORD_JOINS_CHANNEL_ID"))
	rankRoles := rankRoleConfig{
		Elo1000: strings.TrimSpace(os.Getenv("DISCORD_ROLE_ELO_1000_ID")),
		Elo1500: strings.TrimSpace(os.Getenv("DISCORD_ROLE_ELO_1500_ID")),
		Elo2000: strings.TrimSpace(os.Getenv("DISCORD_ROLE_ELO_2000_ID")),
	}
	store, err := persistence.NewFromEnv()
	if err != nil {
		return nil, err
	}
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		store.Close()
		return nil, err
	}
	w := &worker{store: store, guildID: guildID, joinsChannelID: joinsChannelID, rankRoles: rankRoles, session: session}
	session.Identify.Intents = discordgo.IntentsGuildMembers | discordgo.IntentsGuildMessages
	session.AddHandler(w.onGuildMemberAdd)
	session.AddHandler(w.onMessageCreate)
	return w, nil
}

func (w *worker) openDiscord() error {
	if err := w.session.Open(); err != nil {
		return err
	}
	if err := w.ensureRankRoles(); err != nil {
		observability.Log("warn", "discord rank role bootstrap failed", map[string]any{"error": err.Error()})
	}
	w.ready.Store(true)
	return nil
}

func (w *worker) close() {
	if w.session != nil {
		_ = w.session.Close()
	}
	if w.store != nil {
		w.store.Close()
	}
}

func (w *worker) onGuildMemberAdd(_ *discordgo.Session, event *discordgo.GuildMemberAdd) {
	if event == nil || event.User == nil || event.GuildID != w.guildID {
		return
	}
	w.awardDiscordMemberBadge(event.User.ID, "member_add")
	if err := w.syncRankRoles(event.User.ID); err != nil {
		log.Printf("discord rank role sync failed for %s: %v", event.User.ID, err)
	}
}

func (w *worker) onMessageCreate(_ *discordgo.Session, event *discordgo.MessageCreate) {
	if event == nil || event.Message == nil || event.Author == nil {
		return
	}
	if w.joinsChannelID == "" || event.GuildID != w.guildID || event.ChannelID != w.joinsChannelID {
		return
	}
	if event.Type != discordgo.MessageTypeGuildMemberJoin {
		return
	}
	w.awardDiscordMemberBadge(event.Author.ID, "joins_channel")
	if err := w.syncRankRoles(event.Author.ID); err != nil {
		log.Printf("discord rank role sync failed for %s: %v", event.Author.ID, err)
	}
}

func (w *worker) awardDiscordMemberBadge(discordUserID, source string) {
	if awarded, err := w.store.AwardDiscordServerMemberByDiscordID(discordUserID); err != nil {
		log.Printf("discord member badge award failed for %s: %v", discordUserID, err)
	} else if awarded {
		observability.Log("info", "discord member badge awarded", map[string]any{"discordUserId": discordUserID, "source": source})
	}
}

func (w *worker) startReconciliation(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	go func() {
		w.reconcileMembers(ctx)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.reconcileMembers(ctx)
			}
		}
	}()
}

func (w *worker) reconcileMembers(ctx context.Context) {
	after := ""
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		members, err := w.session.GuildMembers(w.guildID, after, 1000)
		if err != nil {
			log.Printf("discord member reconcile failed: %v", err)
			return
		}
		if len(members) == 0 {
			return
		}
		for _, member := range members {
			if member == nil || member.User == nil {
				continue
			}
			after = member.User.ID
			w.awardDiscordMemberBadge(member.User.ID, "reconcile")
			if err := w.syncRankRoles(member.User.ID); err != nil {
				log.Printf("discord member role reconcile failed for %s: %v", member.User.ID, err)
			}
		}
		if len(members) < 1000 {
			return
		}
	}
}

func (w *worker) startDiscordSyncWorker(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(discordSyncInterval)
		defer ticker.Stop()
		for {
			w.drainDiscordSync(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (w *worker) drainDiscordSync(ctx context.Context) {
	for i := 0; i < discordSyncBatch; i++ {
		if ctx.Err() != nil {
			return
		}
		processed, err := w.processOneDiscordSync()
		if err != nil {
			observability.Log("warn", "discord sync processing failed", map[string]any{"error": err.Error()})
			return
		}
		if !processed {
			return
		}
	}
}

func (w *worker) processOneDiscordSync() (bool, error) {
	item, ok, err := w.store.ClaimPendingDiscordSync(time.Now())
	if err != nil || !ok {
		return false, err
	}
	var processErr error
	switch item.Action {
	case persistence.DiscordSyncActionCleanupRoles:
		processErr = w.cleanupRankRoles(item.DiscordUserID)
	case persistence.DiscordSyncActionSync:
		processErr = w.syncDiscordUser(item.DiscordUserID)
	default:
		processErr = errors.New("unknown discord sync action")
	}
	if processErr != nil {
		return true, w.store.MarkDiscordSyncFailed(item.ID, nextDiscordSyncAttempt(item.Attempts), processErr.Error())
	}
	if err := w.store.MarkDiscordSyncProcessed(item.ID); err != nil {
		return true, err
	}
	return true, nil
}

func nextDiscordSyncAttempt(attempts int) time.Time {
	if attempts <= 0 {
		attempts = 1
	}
	delays := []time.Duration{
		15 * time.Second,
		30 * time.Second,
		time.Minute,
		2 * time.Minute,
		5 * time.Minute,
		10 * time.Minute,
	}
	idx := attempts - 1
	if idx >= len(delays) {
		idx = len(delays) - 1
	}
	return time.Now().Add(delays[idx])
}

func (w *worker) syncDiscordUser(discordUserID string) error {
	member, err := w.session.GuildMember(w.guildID, discordUserID)
	if err != nil {
		if isDiscordNotFound(err) {
			return nil
		}
		return err
	}
	if member == nil || member.User == nil {
		return nil
	}
	w.awardDiscordMemberBadge(discordUserID, "sync")
	return w.syncRankRoles(discordUserID)
}

func (w *worker) syncRankRoles(discordUserID string) error {
	user, ok, err := w.store.GetDiscordLinkedUser(discordUserID)
	if err != nil {
		return err
	}
	if !ok {
		return w.cleanupRankRoles(discordUserID)
	}
	member, err := w.session.GuildMember(w.guildID, discordUserID)
	if err != nil {
		if isDiscordNotFound(err) {
			return nil
		}
		return err
	}
	if member == nil {
		return nil
	}
	targetRole := w.rankRoleForMMR(user.HighestEloBadgeMMR)
	return w.applyExclusiveRankRole(discordUserID, member.Roles, targetRole)
}

func (w *worker) cleanupRankRoles(discordUserID string) error {
	member, err := w.session.GuildMember(w.guildID, discordUserID)
	if err != nil {
		if isDiscordNotFound(err) {
			return nil
		}
		return err
	}
	if member == nil {
		return nil
	}
	return w.applyExclusiveRankRole(discordUserID, member.Roles, "")
}

func (w *worker) rankRoleForMMR(mmr int) string {
	switch {
	case mmr >= 2000:
		return w.rankRoles.Elo2000
	case mmr >= 1500:
		return w.rankRoles.Elo1500
	case mmr >= 1000:
		return w.rankRoles.Elo1000
	default:
		return ""
	}
}

func (w *worker) ensureRankRoles() error {
	roles, err := w.session.GuildRoles(w.guildID)
	if err != nil {
		return err
	}
	if w.rankRoles.Elo1000 == "" {
		roleID, err := w.ensureRankRole(roles, "1k", 0x4c9aff)
		if err != nil {
			return err
		}
		w.rankRoles.Elo1000 = roleID
	}
	if w.rankRoles.Elo1500 == "" {
		roleID, err := w.ensureRankRole(roles, "1.5k", 0xffc857)
		if err != nil {
			return err
		}
		w.rankRoles.Elo1500 = roleID
	}
	if w.rankRoles.Elo2000 == "" {
		roleID, err := w.ensureRankRole(roles, "2k", 0xff5c8a)
		if err != nil {
			return err
		}
		w.rankRoles.Elo2000 = roleID
	}
	return nil
}

func (w *worker) ensureRankRole(existing []*discordgo.Role, name string, color int) (string, error) {
	for _, role := range existing {
		if role != nil && role.Name == name {
			return role.ID, nil
		}
	}
	hoist := false
	mentionable := false
	role, err := w.session.GuildRoleCreate(w.guildID, &discordgo.RoleParams{
		Name:        name,
		Color:       &color,
		Hoist:       &hoist,
		Mentionable: &mentionable,
	})
	if err != nil {
		return "", err
	}
	if role == nil {
		return "", errors.New("discord role create returned no role")
	}
	observability.Log("info", "discord rank role created", map[string]any{"roleId": role.ID, "name": name})
	return role.ID, nil
}

func (w *worker) applyExclusiveRankRole(discordUserID string, currentRoles []string, targetRole string) error {
	current := map[string]bool{}
	for _, roleID := range currentRoles {
		current[roleID] = true
	}
	for _, roleID := range []string{w.rankRoles.Elo1000, w.rankRoles.Elo1500, w.rankRoles.Elo2000} {
		if roleID == "" {
			continue
		}
		if roleID == targetRole {
			if !current[roleID] {
				if err := w.session.GuildMemberRoleAdd(w.guildID, discordUserID, roleID); err != nil {
					return err
				}
			}
			continue
		}
		if current[roleID] {
			if err := w.session.GuildMemberRoleRemove(w.guildID, discordUserID, roleID); err != nil {
				return err
			}
		}
	}
	return nil
}

func isDiscordNotFound(err error) bool {
	var restErr *discordgo.RESTError
	if !errors.As(err, &restErr) || restErr.Response == nil {
		return false
	}
	return restErr.Response.StatusCode == http.StatusNotFound
}

func (w *worker) healthLive(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte("ok"))
}

func (w *worker) healthReady(rw http.ResponseWriter, _ *http.Request) {
	if w.draining.Load() || !w.ready.Load() {
		http.Error(rw, "not ready", http.StatusServiceUnavailable)
		return
	}
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte("ready"))
}

func handleWorkerShutdown(w *worker, srv *http.Server, cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sigCh)

	<-sigCh
	w.draining.Store(true)
	cancel()
	time.Sleep(workerDrainTimeout)

	ctx, shutdownCancel := context.WithTimeout(context.Background(), workerShutdownTimeout)
	defer shutdownCancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("discord worker shutdown failed: %v", err)
	}
}

func getenv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

func getenvDuration(k string, fallback time.Duration) time.Duration {
	if value := strings.TrimSpace(os.Getenv(k)); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return fallback
}
