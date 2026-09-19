package gameplay

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// loadCases reads the deterministic coordinate corpus shared with any other
// implementations that need identical geographic inputs.
func loadCases(t *testing.T) []GeoCase {
	t.Helper()
	path := filepath.Join("..", "..", "..", "tests", "shared", "geo-cases.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read shared geo cases: %v", err)
	}
	var wrapper struct {
		Cases []GeoCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		t.Fatalf("parse shared geo cases: %v", err)
	}
	return wrapper.Cases
}

type GeoCase struct {
	Lat1 float64 `json:"lat1"`
	Lng1 float64 `json:"lng1"`
	Lat2 float64 `json:"lat2"`
	Lng2 float64 `json:"lng2"`
}

const maxHaversineKm = 6371.0 * math.Pi

func distanceTolerance() float64 { return 1e-6 }

func TestHaversineKmGeneratedCases(t *testing.T) {
	for _, c := range loadCases(t) {
		d1 := HaversineKm(c.Lat1, c.Lng1, c.Lat2, c.Lng2)
		d2 := HaversineKm(c.Lat2, c.Lng2, c.Lat1, c.Lng1)
		if math.IsNaN(d1) || math.IsInf(d1, 0) {
			t.Fatalf("non-finite distance for %+v: %v", c, d1)
		}
		if d1 < 0 || d1 > maxHaversineKm+distanceTolerance() {
			t.Fatalf("distance out of bounds for %+v: %v", c, d1)
		}
		if math.Abs(d1-d2) > distanceTolerance() {
			t.Fatalf("asymmetric distance for %+v: %v vs %v", c, d1, d2)
		}
	}
}

func TestHaversineKmBoundaryCases(t *testing.T) {
	tol := distanceTolerance()
	selfDistances := [][2]float64{
		{0, 0},
		{90, 0}, {-90, 0},
		{52.52, 13.405},
		{-41.3, 174.78},
		{89.9999, 179.9999},
	}
	for _, p := range selfDistances {
		if d := HaversineKm(p[0], p[1], p[0], p[1]); math.Abs(d) > tol {
			t.Fatalf("self-distance at %+v should be zero, got %v", p, d)
		}
	}

	// Pole pairs: any two points on the same pole coincide; opposite poles are
	// half the earth's circumference apart.
	if d := HaversineKm(90, 0, 90, 137); math.Abs(d) > tol {
		t.Fatalf("same-pole distance should be zero, got %v", d)
	}
	if d := HaversineKm(90, 0, -90, 0); math.Abs(d-maxHaversineKm) > tol {
		t.Fatalf("pole-to-pole distance should be %v, got %v", maxHaversineKm, d)
	}

	// Date-line crossing: a short hop across ±180° must stay short, not wrap
	// around the planet.
	if d := HaversineKm(0, 179.9, 0, -179.9); d > 100 {
		t.Fatalf("date-line hop should be short, got %v km", d)
	}

	// Near-antipodal points approach (but never exceed) the maximum.
	nearAntipodal := HaversineKm(0, 0, 0.5, 179.9)
	if nearAntipodal < maxHaversineKm-500 || nearAntipodal > maxHaversineKm {
		t.Fatalf("near-antipodal distance should be close to %v, got %v", maxHaversineKm, nearAntipodal)
	}
	if d := HaversineKm(0, 0, 0, 180); math.Abs(d-maxHaversineKm) > tol {
		t.Fatalf("exact antipodal distance should be %v, got %v", maxHaversineKm, d)
	}

	// Equator quarter-circumference sanity check (90° of longitude).
	if d := HaversineKm(0, 0, 0, 90); math.Abs(d-maxHaversineKm/2) > tol {
		t.Fatalf("equatorial 90° distance should be %v, got %v", maxHaversineKm/2, d)
	}
}

func TestClampDistanceKm(t *testing.T) {
	cases := []struct{ in, want float64 }{
		{-5, 0},
		{0, 0},
		{42, 42},
		{maxHaversineKm + 100, maxHaversineKm},
	}
	for _, c := range cases {
		if got := ClampDistanceKm(c.in); math.Abs(got-c.want) > 1e-9 {
			t.Fatalf("ClampDistanceKm(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestRoundScoreBoundedAndMonotonic(t *testing.T) {
	for _, c := range loadCases(t) {
		d := HaversineKm(c.Lat1, c.Lng1, c.Lat2, c.Lng2)
		score := RoundScore(d)
		if score < 0 || score > MaxScore {
			t.Fatalf("score out of bounds for %+v (d=%v): %d", c, d, score)
		}
	}

	// Monotonic non-increasing in distance.
	prev := math.MaxInt
	for km := 0.0; km <= maxHaversineKm+1; km += 25 {
		score := RoundScore(km)
		if score > prev {
			t.Fatalf("score increased with distance: at %v km score %d after %d", km, score, prev)
		}
		prev = score
	}

	// Within the perfect-guess radius the score is maximal.
	if score := RoundScore(perfectGuess - 0.01); score != MaxScore {
		t.Fatalf("score within perfect radius should be %d, got %d", MaxScore, score)
	}
	if score := RoundScore(perfectGuess); score != MaxScore {
		t.Fatalf("score at perfect threshold should be %d, got %d", MaxScore, score)
	}

	// Just outside the perfect radius the score must drop below the maximum.
	if score := RoundScore(perfectGuess + 0.5); score >= MaxScore {
		t.Fatalf("score just outside perfect radius should drop below %d, got %d", MaxScore, score)
	}

	// Maximum distance scores essentially nothing but never goes negative.
	if score := RoundScore(maxHaversineKm); score < 0 || score > 10 {
		t.Fatalf("antipodal score should be near zero, got %d", score)
	}
}
