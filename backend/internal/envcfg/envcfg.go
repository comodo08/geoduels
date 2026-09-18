// Package envcfg centralizes the environment-variable parsing helpers that
// every service previously duplicated.
package envcfg

import (
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func Get(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

func Duration(k string, fallback time.Duration) time.Duration {
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

func Int(k string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func Bool(k string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return fallback
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func SameSite(k string, fallback http.SameSite) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(k))) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	case "lax":
		return http.SameSiteLaxMode
	case "":
		return fallback
	default:
		return fallback
	}
}

func RequiredSecret(k string, minLen int) ([]byte, error) {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return nil, errors.New(k + " is required")
	}
	if len(v) < minLen {
		return nil, errors.New(k + " must be at least " + strconv.Itoa(minLen) + " characters")
	}
	return []byte(v), nil
}
