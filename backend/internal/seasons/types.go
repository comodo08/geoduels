package seasons

import "time"

type RankedSeasonSettings struct {
	ActiveSeasonID  string     `json:"activeSeasonId"`
	MonthlyResetDay int        `json:"monthlyResetDay"`
	NextResetAt     *time.Time `json:"nextResetAt,omitempty"`
	LastResetAt     *time.Time `json:"lastResetAt,omitempty"`
}

type RankedSeasonResetResult struct {
	PreviousSeasonID string `json:"previousSeasonId"`
	ActiveSeasonID   string `json:"activeSeasonId"`
	PlayersSeeded    int    `json:"playersSeeded"`
	ResetAt          string `json:"resetAt"`
}
