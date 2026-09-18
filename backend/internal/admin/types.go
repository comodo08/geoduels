package admin

import "geoduels/pkg/contracts"

type AdminPlayerSummary = contracts.AdminPlayerSummary
type UserRoleGrant = contracts.UserRoleGrant

type AdminPlayerStats struct {
	TotalMatches     int `json:"totalMatches"`
	RankedMatches    int `json:"rankedMatches"`
	DuelMatches      int `json:"duelMatches"`
	SingleplayerRuns int `json:"singleplayerRuns"`
	Wins             int `json:"wins"`
	Losses           int `json:"losses"`
}

type AdminPlayerDetail struct {
	Player AdminPlayerSummary `json:"player"`
}
