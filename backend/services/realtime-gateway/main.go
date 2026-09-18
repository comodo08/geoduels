package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"

	"geoduels/pkg/coordinator"
	"geoduels/pkg/observability"
)

type realtimeGateway struct {
	state         *coordinator.Store
	redis         *redis.Client
	metrics       *observability.APIMetrics
	draining      atomic.Bool
	activeSockets atomic.Int64
	drainTTL      time.Duration
}

var wsProxyUpgrader = websocket.Upgrader{CheckOrigin: wsOriginAllowed}

func main() {
	rdb, redisCleanup, err := redisFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	g := &realtimeGateway{
		state:    coordinator.NewStore(rdb, getenvDuration("GAMEPLAY_NODE_TTL", 10*time.Second), 2*time.Hour, 24*time.Hour, 5*time.Second),
		redis:    rdb,
		metrics:  observability.NewAPIMetrics(),
		drainTTL: getenvDuration("REALTIME_GATEWAY_DRAIN_TIMEOUT", 18*time.Minute),
	}
	defer redisCleanup()

	r := mux.NewRouter()
	r.HandleFunc("/health", g.healthLive).Methods(http.MethodGet)
	r.HandleFunc("/health/live", g.healthLive).Methods(http.MethodGet)
	r.HandleFunc("/health/ready", g.healthReady).Methods(http.MethodGet)
	r.HandleFunc("/ws/{node}", g.wsProxy).Methods(http.MethodGet)
	r.Handle("/metrics", observability.Handler(g.metrics.Registry)).Methods(http.MethodGet)

	addr := getenv("REALTIME_GATEWAY_ADDR", ":8092")
	srv := &http.Server{
		Addr:              addr,
		Handler:           cors(g.metrics.Middleware(r)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	observability.Log("info", "realtime-gateway startup", map[string]any{"addr": addr})
	go g.handleShutdown(srv)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func (g *realtimeGateway) wsProxy(w http.ResponseWriter, r *http.Request) {
	if g.draining.Load() {
		http.Error(w, "draining", http.StatusServiceUnavailable)
		return
	}
	nodeRoute := strings.TrimSpace(mux.Vars(r)["node"])
	if nodeRoute == "" {
		http.Error(w, "missing node", http.StatusBadRequest)
		return
	}
	node, ok, err := g.state.GetNodeByRoute(r.Context(), nodeRoute)
	if err != nil {
		http.Error(w, "routing unavailable", http.StatusBadGateway)
		return
	}
	if !ok || strings.TrimSpace(node.InternalURL) == "" {
		http.Error(w, "node unavailable", http.StatusNotFound)
		return
	}

	target := websocketTarget(node.InternalURL, r.URL.RequestURI())
	backendConn, resp, err := websocket.DefaultDialer.DialContext(r.Context(), target, nil)
	if err != nil {
		if resp != nil && resp.StatusCode > 0 {
			http.Error(w, "gameplay unavailable", resp.StatusCode)
			return
		}
		http.Error(w, "gameplay unavailable", http.StatusBadGateway)
		return
	}
	defer backendConn.Close()

	clientConn, err := wsProxyUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer clientConn.Close()
	g.activeSockets.Add(1)
	defer g.activeSockets.Add(-1)

	var once sync.Once
	closeBoth := func() {
		once.Do(func() {
			_ = clientConn.Close()
			_ = backendConn.Close()
		})
	}

	errc := make(chan error, 2)
	go proxyWS(errc, closeBoth, clientConn, backendConn)
	go proxyWS(errc, closeBoth, backendConn, clientConn)

	if err := <-errc; err != nil && !isExpectedWSClose(err) {
		log.Printf("websocket proxy failed for route %s: %v", nodeRoute, err)
	}
	closeBoth()
	<-errc
}

func (g *realtimeGateway) healthLive(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (g *realtimeGateway) healthReady(w http.ResponseWriter, _ *http.Request) {
	if g.draining.Load() {
		http.Error(w, "draining", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := g.redis.Ping(ctx).Err(); err != nil {
		http.Error(w, "redis not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}

func (g *realtimeGateway) handleShutdown(srv *http.Server) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sigCh)

	<-sigCh
	g.draining.Store(true)
	deadline := time.Now().Add(g.drainTTL)
	for time.Now().Before(deadline) {
		if g.activeSockets.Load() == 0 {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("realtime-gateway shutdown failed: %v", err)
	}
}

func websocketTarget(baseURL, requestURI string) string {
	base := strings.TrimRight(baseURL, "/")
	switch {
	case strings.HasPrefix(base, "https://"):
		base = "wss://" + strings.TrimPrefix(base, "https://")
	case strings.HasPrefix(base, "http://"):
		base = "ws://" + strings.TrimPrefix(base, "http://")
	case strings.HasPrefix(base, "wss://"), strings.HasPrefix(base, "ws://"):
	default:
		base = "ws://" + base
	}
	return base + requestURI
}

func proxyWS(errc chan<- error, closeBoth func(), dst, src *websocket.Conn) {
	for {
		messageType, payload, err := src.ReadMessage()
		if err != nil {
			closeBoth()
			errc <- err
			return
		}
		if err := dst.WriteMessage(messageType, payload); err != nil {
			closeBoth()
			errc <- err
			return
		}
	}
}

func isExpectedWSClose(err error) bool {
	if err == nil || errors.Is(err, io.EOF) {
		return true
	}
	return websocket.IsCloseError(
		err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseNoStatusReceived,
	)
}

func cors(next http.Handler) http.Handler {
	allowed := allowedOriginsSet()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && (allowed["*"] || allowed[origin]) {
			if allowed["*"] {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func wsOriginAllowed(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	allowed := allowedOriginsSet()
	return allowed["*"] || allowed[origin]
}

func allowedOriginsSet() map[string]bool {
	raw := getenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000")
	out := map[string]bool{}
	for _, s := range strings.Split(raw, ",") {
		origin := strings.TrimSpace(s)
		if origin == "" {
			continue
		}
		out[origin] = true
	}
	return out
}

func getenv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

func getenvDuration(k string, fallback time.Duration) time.Duration {
	v := os.Getenv(k)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func redisFromEnv() (*redis.Client, func(), error) {
	url := getenv("REDIS_URL", "")
	if url == "" {
		return nil, nil, errors.New("REDIS_URL is required")
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, nil, err
	}
	rdb := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, nil, err
	}
	return rdb, func() { _ = rdb.Close() }, nil
}
