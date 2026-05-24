package observability

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type RuntimeMetrics struct {
	Registry *prometheus.Registry

	CommandTotal           *prometheus.CounterVec
	CommandLatencySeconds  *prometheus.HistogramVec
	OwnershipTransfers     prometheus.Counter
	OwnershipRenewFailures prometheus.Counter
	ForwardTotal           *prometheus.CounterVec
	ResumeFailures         prometheus.Counter
	NATSPublishFailures    prometheus.Counter
	DBWriteFailures        prometheus.Counter
	QueueWaitSeconds       prometheus.Histogram
	ConnectedUsers         prometheus.Gauge
}

func NewRuntimeMetrics() *RuntimeMetrics {
	r := prometheus.NewRegistry()
	f := promauto.With(r)
	return &RuntimeMetrics{
		Registry: r,
		CommandTotal: f.NewCounterVec(prometheus.CounterOpts{
			Name: "geoduels_runtime_commands_total",
			Help: "Total runtime commands processed",
		}, []string{"type", "status", "error_code"}),
		CommandLatencySeconds: f.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "geoduels_runtime_command_latency_seconds",
			Help:    "Runtime command processing latency",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.5, 1, 2},
		}, []string{"type"}),
		OwnershipTransfers: f.NewCounter(prometheus.CounterOpts{
			Name: "geoduels_runtime_ownership_transfers_total",
			Help: "Number of ownership transfers/acquisitions",
		}),
		OwnershipRenewFailures: f.NewCounter(prometheus.CounterOpts{
			Name: "geoduels_runtime_ownership_renew_failures_total",
			Help: "Failed ownership renewals",
		}),
		ForwardTotal: f.NewCounterVec(prometheus.CounterOpts{
			Name: "geoduels_runtime_forward_total",
			Help: "Forwarded command count",
		}, []string{"result"}),
		ResumeFailures: f.NewCounter(prometheus.CounterOpts{
			Name: "geoduels_runtime_resume_failures_total",
			Help: "Failed resume attempts",
		}),
		NATSPublishFailures: f.NewCounter(prometheus.CounterOpts{
			Name: "geoduels_runtime_nats_publish_failures_total",
			Help: "Failed NATS publishes",
		}),
		DBWriteFailures: f.NewCounter(prometheus.CounterOpts{
			Name: "geoduels_runtime_db_write_failures_total",
			Help: "Failed runtime DB writes",
		}),
		QueueWaitSeconds: f.NewHistogram(prometheus.HistogramOpts{
			Name:    "geoduels_runtime_queue_wait_seconds",
			Help:    "Estimated queue wait durations",
			Buckets: []float64{0.5, 1, 2, 5, 10, 20, 40, 60, 120},
		}),
		ConnectedUsers: f.NewGauge(prometheus.GaugeOpts{
			Name: "geoduels_runtime_connected_users",
			Help: "Current connected users on this runtime instance",
		}),
	}
}

type APIMetrics struct {
	Registry *prometheus.Registry
	Requests *prometheus.CounterVec
	Latency  *prometheus.HistogramVec
}

func NewAPIMetrics() *APIMetrics {
	r := prometheus.NewRegistry()
	f := promauto.With(r)
	return &APIMetrics{
		Registry: r,
		Requests: f.NewCounterVec(prometheus.CounterOpts{
			Name: "geoduels_api_requests_total",
			Help: "API request counts",
		}, []string{"path", "method", "status"}),
		Latency: f.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "geoduels_api_request_latency_seconds",
			Help:    "API request latency",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.5, 1, 2},
		}, []string{"path", "method"}),
	}
}

func Handler(reg *prometheus.Registry) http.Handler {
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}

func (m *APIMetrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		path := requestMetricPath(next, r)
		m.Requests.WithLabelValues(path, r.Method, statusCode(rw.status)).Inc()
		m.Latency.WithLabelValues(path, r.Method).Observe(time.Since(start).Seconds())
	})
}

func requestMetricPath(next http.Handler, r *http.Request) string {
	if route := mux.CurrentRoute(r); route != nil {
		if template, err := route.GetPathTemplate(); err == nil && template != "" {
			return template
		}
	}
	if router, ok := next.(*mux.Router); ok {
		var match mux.RouteMatch
		if router.Match(r, &match) && match.Route != nil {
			if template, err := match.Route.GetPathTemplate(); err == nil && template != "" {
				return template
			}
		}
	}
	if r.URL != nil && r.URL.Path != "" {
		return r.URL.Path
	}
	return "unmatched"
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(statusCode int) {
	s.status = statusCode
	s.ResponseWriter.WriteHeader(statusCode)
}

func (s *statusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := s.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return h.Hijack()
}

func (s *statusRecorder) Push(target string, opts *http.PushOptions) error {
	p, ok := s.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return p.Push(target, opts)
}

func (s *statusRecorder) ReadFrom(r io.Reader) (int64, error) {
	rf, ok := s.ResponseWriter.(io.ReaderFrom)
	if !ok {
		return io.Copy(s.ResponseWriter, r)
	}
	return rf.ReadFrom(r)
}

func statusCode(code int) string {
	return http.StatusText(code)
}
