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
)

type worker struct {
	store    persistence.Store
	guildID  string
	session  *discordgo.Session
	draining atomic.Bool
	ready    atomic.Bool
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
	store, err := persistence.NewFromEnv()
	if err != nil {
		return nil, err
	}
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		store.Close()
		return nil, err
	}
	w := &worker{store: store, guildID: guildID, session: session}
	session.Identify.Intents = discordgo.IntentsGuildMembers
	session.AddHandler(w.onGuildMemberAdd)
	return w, nil
}

func (w *worker) openDiscord() error {
	if err := w.session.Open(); err != nil {
		return err
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
	if awarded, err := w.store.AwardDiscordServerMemberByDiscordID(event.User.ID); err != nil {
		log.Printf("discord member badge award failed for %s: %v", event.User.ID, err)
	} else if awarded {
		observability.Log("info", "discord member badge awarded", map[string]any{"discordUserId": event.User.ID})
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
			if _, err := w.store.AwardDiscordServerMemberByDiscordID(member.User.ID); err != nil {
				log.Printf("discord member badge reconcile failed for %s: %v", member.User.ID, err)
			}
		}
		if len(members) < 1000 {
			return
		}
	}
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
