package maps

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"

	"geoduels/pkg/mapthumbnails"
)

// mapRow is the compact in-memory form of one map location.
type mapRow struct {
	Lat         float64
	Lng         float64
	Country     string
	PanoID      *string
	Heading     *float64
	Pitch       *float64
	LatE7       int32
	LngE7       int32
	HeadingCDeg *int16
	PitchCDeg   *int16
	RandKey     int32
}

func stableRand(lat, lng float64) int32 {
	h := sha1.Sum([]byte(fmt.Sprintf("%.8f:%.8f", lat, lng)))
	v := int(h[0])<<16 | int(h[1])<<8 | int(h[2])
	return int32(v)
}

func compactMapRow(lat, lng float64) mapRow {
	return mapRow{
		Lat:     lat,
		Lng:     lng,
		LatE7:   int32(math.Round(lat * 10_000_000)),
		LngE7:   int32(math.Round(lng * 10_000_000)),
		RandKey: stableRand(lat, lng),
	}
}

func compactAngle(value float64, pitch bool) int16 {
	if pitch {
		value = math.Max(-90, math.Min(90, value))
	} else {
		value = math.Mod(value+180, 360)
		if value < 0 {
			value += 360
		}
		value -= 180
	}
	return int16(math.Round(value * 100))
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
		row := compactMapRow(lat, lng)
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
			value := compactAngle(heading, false)
			row.HeadingCDeg = &value
		}
		if pitch, ok := asFloat(it["pitch"]); ok {
			row.Pitch = &pitch
			value := compactAngle(pitch, true)
			row.PitchCDeg = &value
		}
		out = append(out, row)
	}
	return out, nil
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

func decodeMapRows(source io.Reader, maxLocations int) ([]mapRow, string, int, error) {
	hasher := sha256.New()
	decoder := json.NewDecoder(io.TeeReader(source, hasher))
	raw, err := decodeRawMapLocations(decoder, maxLocations)
	if err != nil {
		return nil, "", 0, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != nil && !errors.Is(err, io.EOF) {
		return nil, "", 0, err
	} else if err == nil {
		return nil, "", 0, errors.New("map JSON must contain one top-level value")
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
		parsed := compactMapRow(*row.Lat, *row.Lng)
		parsed.Country = country
		parsed.PanoID = pano
		parsed.Heading = row.Heading
		parsed.Pitch = row.Pitch
		if row.Heading != nil {
			value := compactAngle(*row.Heading, false)
			parsed.HeadingCDeg = &value
		}
		if row.Pitch != nil {
			value := compactAngle(*row.Pitch, true)
			parsed.PitchCDeg = &value
		}
		out = append(out, parsed)
	}
	return out, hex.EncodeToString(hasher.Sum(nil)), rejected, nil
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

func decodeRawMapLocations(decoder *json.Decoder, maxLocations int) ([]rawMapLocation, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil, errors.New("map JSON must be an array or include customCoordinates")
	}
	switch delim {
	case '[':
		return decodeRawMapArray(decoder, maxLocations)
	case '{':
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			key, _ := keyToken.(string)
			if key != "customCoordinates" {
				var discard json.RawMessage
				if err := decoder.Decode(&discard); err != nil {
					return nil, err
				}
				continue
			}
			arrayToken, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			if arrayToken != json.Delim('[') {
				return nil, errors.New("customCoordinates must be an array")
			}
			rows, err := decodeRawMapArray(decoder, maxLocations)
			if err != nil {
				return nil, err
			}
			for decoder.More() {
				if _, err := decoder.Token(); err != nil {
					return nil, err
				}
				var discard json.RawMessage
				if err := decoder.Decode(&discard); err != nil {
					return nil, err
				}
			}
			if _, err := decoder.Token(); err != nil {
				return nil, err
			}
			return rows, nil
		}
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		return nil, errors.New("map JSON must include customCoordinates")
	default:
		return nil, errors.New("map JSON must be an array or include customCoordinates")
	}
}

func decodeRawMapArray(decoder *json.Decoder, maxLocations int) ([]rawMapLocation, error) {
	if maxLocations <= 0 || maxLocations > absoluteMaxMapLocations {
		maxLocations = absoluteMaxMapLocations
	}
	rows := make([]rawMapLocation, 0, 4096)
	for decoder.More() {
		if len(rows) >= maxLocations {
			return nil, fmt.Errorf("map limit is %d locations", maxLocations)
		}
		var row rawMapLocation
		if err := decoder.Decode(&row); err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	return rows, nil
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
	if mapthumbnails.ValidKey(v) {
		return v
	}
	return fmt.Sprintf("generic/variant-%d", normalizeThumbnailVariant(fallbackVariant))
}
