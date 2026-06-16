package persistence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

func decodeMapRows(source io.Reader) ([]mapRow, string, int, error) {
	b, err := io.ReadAll(source)
	if err != nil {
		return nil, "", 0, err
	}
	sum := sha256.Sum256(b)
	raw, err := parseRawMapLocations(b)
	if err != nil {
		return nil, "", 0, err
	}
	if len(raw) > maxMapLocations {
		return nil, "", 0, fmt.Errorf("map limit is %d locations", maxMapLocations)
	}

	out := make([]mapRow, 0, len(raw))
	panos := map[string]struct{}{}
	coords := map[string]struct{}{}
	rejected := 0
	for _, row := range raw {
		if row.Lat == nil || row.Lng == nil || *row.Lat < -90 || *row.Lat > 90 || *row.Lng < -180 || *row.Lng > 180 {
			rejected++
			continue
		}
		coord := fmt.Sprintf("%.8f:%.8f", *row.Lat, *row.Lng)
		if _, ok := coords[coord]; ok {
			rejected++
			continue
		}
		panoID := strings.TrimSpace(row.PanoID)
		if panoID == "" {
			panoID = strings.TrimSpace(row.Extra.PanoID)
		}
		if panoID != "" {
			if _, ok := panos[panoID]; ok {
				rejected++
				continue
			}
			panos[panoID] = struct{}{}
		}
		coords[coord] = struct{}{}
		country := strings.TrimSpace(row.Country)
		if country == "" {
			country = strings.TrimSpace(row.CountryCode)
		}
		if len(country) > 80 {
			country = country[:80]
		}
		var pano *string
		if panoID != "" {
			if len(panoID) > 255 {
				rejected++
				continue
			}
			pano = &panoID
		}
		out = append(out, mapRow{Lat: *row.Lat, Lng: *row.Lng, Country: country, PanoID: pano, Heading: row.Heading, Pitch: row.Pitch, RandKey: stableRand(*row.Lat, *row.Lng)})
	}
	return out, hex.EncodeToString(sum[:]), rejected, nil
}

type rawMapLocation struct {
	Lat         *float64 `json:"lat"`
	Lng         *float64 `json:"lng"`
	Country     string   `json:"country"`
	CountryCode string   `json:"countryCode"`
	PanoID      string   `json:"panoId"`
	Heading     *float64 `json:"heading"`
	Pitch       *float64 `json:"pitch"`
	Extra       struct {
		PanoID string `json:"panoId"`
	} `json:"extra"`
}

func parseRawMapLocations(b []byte) ([]rawMapLocation, error) {
	var rows []rawMapLocation
	if err := json.Unmarshal(b, &rows); err == nil {
		return rows, nil
	}

	var envelope struct {
		CustomCoordinates []rawMapLocation `json:"customCoordinates"`
	}
	if err := json.Unmarshal(b, &envelope); err != nil {
		return nil, err
	}
	if envelope.CustomCoordinates == nil {
		return nil, errors.New("map JSON must be an array or include customCoordinates")
	}
	return envelope.CustomCoordinates, nil
}

func normalizeMapVisibility(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "public":
		return "public"
	case "unlisted":
		return "unlisted"
	default:
		return "private"
	}
}

func normalizeMapScope(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "official", "community", "favorites", "mine":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return ""
	}
}

func normalizeMapSort(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "popular", "new":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "trending"
	}
}

func mapSearchPattern(v string) string {
	term := strings.Join(strings.Fields(strings.TrimSpace(v)), " ")
	if term == "" {
		return ""
	}
	runes := []rune(term)
	if len(runes) > 80 {
		term = string(runes[:80])
	}
	var b strings.Builder
	b.Grow(len(term) + 2)
	b.WriteByte('%')
	for _, r := range term {
		switch r {
		case '\\', '%', '_':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	b.WriteByte('%')
	return b.String()
}

func normalizeMapDifficulty(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "easy", "hard":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "normal"
	}
}

func normalizeThumbnailVariant(v int) int {
	if v < 1 || v > 5 {
		return 1
	}
	return v
}

func normalizeThumbnailKey(v string, fallbackVariant int) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = strings.Trim(v, "/")
	if validMapThumbnailKey(v) {
		return v
	}
	return fmt.Sprintf("generic/variant-%d", normalizeThumbnailVariant(fallbackVariant))
}

func validMapThumbnailKey(v string) bool {
	if strings.HasPrefix(v, "generic/variant-") {
		switch strings.TrimPrefix(v, "generic/variant-") {
		case "1", "2", "3", "4", "5":
			return true
		}
	}
	switch v {
	case "continents/africa", "continents/antarctica", "continents/asia", "continents/europe", "continents/north-america", "continents/oceania", "continents/south-america":
		return true
	default:
		return validCountryThumbnailKey(v)
	}
}

func validCountryThumbnailKey(v string) bool {
	slug, ok := strings.CutPrefix(v, "countries/")
	if !ok || slug == "" || len(slug) > 80 {
		return false
	}
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return false
	}
	return !strings.Contains(slug, "--") && slug[0] != '-' && slug[len(slug)-1] != '-'
}
