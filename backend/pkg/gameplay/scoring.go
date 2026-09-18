package gameplay

import "math"

const (
	MaxScore      = 5000
	earthRadiusKm = 6371.0
	maxDistanceKm = earthRadiusKm * math.Pi
	perfectGuess  = 0.15
	scoreDecayKm  = 1494.52
	scoreFloor    = 0.033
)

func HaversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	latRad1 := lat1 * math.Pi / 180.0
	latRad2 := lat2 * math.Pi / 180.0
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(latRad1)*math.Cos(latRad2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

func ClampDistanceKm(distanceKm float64) float64 {
	return math.Min(maxDistanceKm, math.Max(0, distanceKm))
}

func RoundScore(distanceKm float64) int {
	clamped := ClampDistanceKm(distanceKm)
	if clamped <= perfectGuess {
		return MaxScore
	}
	return int(math.Floor((MaxScore * math.Exp(-(clamped / scoreDecayKm))) + scoreFloor))
}
