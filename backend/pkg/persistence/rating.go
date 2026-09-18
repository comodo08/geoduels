package persistence

import (
	"time"

	"geoduels/pkg/rating"
)

const (
	initialMMR       = rating.InitialMMR
	initialRatingRD  = rating.InitialRatingRD
	minimumRatingRD  = rating.MinimumRatingRD
	maximumRatingRD  = rating.MaximumRatingRD
	minimumRankedMMR = rating.MinimumRankedMMR
	maxDuelMMRDelta  = rating.MaxDuelMMRDelta
)

type RatingState = rating.State
type RatingUpdate = rating.Update

func CalculateDuelRatingUpdates(p1, p2 RatingState, winner string, now time.Time) (RatingUpdate, RatingUpdate) {
	return rating.CalculateDuelUpdates(p1, p2, winner, now)
}

func CalculateDuelDeltas(p1MMR, p2MMR int, p1RD, p2RD float64, winner string) (int, int) {
	return rating.CalculateDuelDeltas(p1MMR, p2MMR, p1RD, p2RD, winner)
}

func inflateRatingRD(rd float64, lastUpdated, now time.Time) float64 {
	return rating.InflateRD(rd, lastUpdated, now)
}

func clampRatingRD(rd float64) float64 {
	return rating.ClampRD(rd)
}

func clampRankedMMR(mmr int) int {
	return rating.ClampRankedMMR(mmr)
}

func capRatingDelta(current, next int) int {
	return rating.CapDelta(current, next)
}
