package persistence

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func getenvInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

type mapRow struct {
	Lat     float64
	Lng     float64
	Country string
	PanoID  *string
	Heading *float64
	Pitch   *float64
	RandKey float64
}

func parseMapRows(b []byte) ([]mapRow, error) {
	var raw []map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		var envelope struct {
			CustomCoordinates []map[string]any `json:"customCoordinates"`
		}
		if err := json.Unmarshal(b, &envelope); err != nil {
			return nil, err
		}
		if envelope.CustomCoordinates == nil {
			return nil, errors.New("map JSON must be an array or include customCoordinates")
		}
		raw = envelope.CustomCoordinates
	}
	out := make([]mapRow, 0, len(raw))
	for _, it := range raw {
		lat, ok1 := asFloat(it["lat"])
		lng, ok2 := asFloat(it["lng"])
		if !ok1 || !ok2 {
			continue
		}
		if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
			continue
		}
		row := mapRow{Lat: lat, Lng: lng, RandKey: stableRand(lat, lng)}
		if country, ok := it["country"].(string); ok {
			row.Country = country
		} else if country, ok := it["countryCode"].(string); ok {
			row.Country = country
		}
		panoID, _ := it["panoId"].(string)
		if strings.TrimSpace(panoID) == "" {
			if extra, ok := it["extra"].(map[string]any); ok {
				panoID, _ = extra["panoId"].(string)
			}
		}
		if panoID = strings.TrimSpace(panoID); panoID != "" {
			row.PanoID = &panoID
		}
		if heading, ok := asFloat(it["heading"]); ok {
			row.Heading = &heading
		}
		if pitch, ok := asFloat(it["pitch"]); ok {
			row.Pitch = &pitch
		}
		out = append(out, row)
	}
	return out, nil
}

func stableRand(lat, lng float64) float64 {
	h := sha1.Sum([]byte(fmt.Sprintf("%.8f:%.8f", lat, lng)))
	v := int(h[0])<<16 | int(h[1])<<8 | int(h[2])
	return float64(v) / float64(1<<24)
}

func asFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	default:
		return 0, false
	}
}

func nullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func newUserID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "u_" + hex.EncodeToString(b)
}

func newDebugMatchID(index int) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("debug-report-%s-%02d", hex.EncodeToString(b), index)
}

func normalizeDBURLForContainer(dsn string) string {
	if _, err := os.Stat("/.dockerenv"); err != nil {
		return dsn
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}
	if u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost" {
		port := u.Port()
		if port == "" {
			port = "5432"
		}
		u.Host = "host.docker.internal:" + port
		return u.String()
	}
	return dsn
}
