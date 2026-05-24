package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestAPIMetricsUsesMuxRouteTemplate(t *testing.T) {
	metrics := NewAPIMetrics()
	r := mux.NewRouter()
	r.HandleFunc("/v1/matches/{id}/session", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods(http.MethodGet)

	req := httptest.NewRequest(http.MethodGet, "/v1/matches/m-123/session", nil)
	metrics.Middleware(r).ServeHTTP(httptest.NewRecorder(), req)

	rec := httptest.NewRecorder()
	Handler(metrics.Registry).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()
	if !strings.Contains(body, `path="/v1/matches/{id}/session"`) {
		t.Fatalf("expected route template metric, got:\n%s", body)
	}
	if strings.Contains(body, `path="/v1/matches/m-123/session"`) {
		t.Fatalf("raw path leaked into metric labels:\n%s", body)
	}
}
