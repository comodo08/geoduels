package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestAPIMetricsUsesEchoRouteTemplate(t *testing.T) {
	metrics := NewAPIMetrics()
	e := echo.New()
	e.HideBanner = true
	e.GET("/v1/matches/:id/session", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}, metrics.EchoMiddleware)

	req := httptest.NewRequest(http.MethodGet, "/v1/matches/m-123/session", nil)
	e.ServeHTTP(httptest.NewRecorder(), req)

	rec := httptest.NewRecorder()
	Handler(metrics.Registry).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()
	if !strings.Contains(body, `path="/v1/matches/:id/session"`) {
		t.Fatalf("expected route template metric, got:\n%s", body)
	}
	if strings.Contains(body, `path="/v1/matches/m-123/session"`) {
		t.Fatalf("raw path leaked into metric labels:\n%s", body)
	}
}
