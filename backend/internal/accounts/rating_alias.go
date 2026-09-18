package accounts

import (
	"time"

	"geoduels/internal/rating"
)

const (
	initialMMR       = rating.InitialMMR
	initialRatingRD  = rating.InitialRatingRD
	minimumRatingRD  = rating.MinimumRatingRD
	maximumRatingRD  = rating.MaximumRatingRD
	minimumRankedMMR = rating.MinimumRankedMMR
	maxDuelMMRDelta  = rating.MaxDuelMMRDelta
	modeDuel         = "duel"
)

type RatingState = rating.State
type RatingUpdate = rating.Update

func CalculateDuelRatingUpdates(p1, p2 RatingState, winner string, now time.Time) (RatingUpdate, RatingUpdate) {
	return rating.CalculateDuelUpdates(p1, p2, winner, now)
}

func clampRatingRD(rd float64) float64     { return rating.ClampRD(rd) }
func capRatingDelta(current, next int) int { return rating.CapDelta(current, next) }
func clampRankedMMR(mmr int) int           { return rating.ClampRankedMMR(mmr) }
