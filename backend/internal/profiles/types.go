package profiles

import "geoduels/pkg/contracts"

type Profile struct {
	UserID            string                  `json:"userId"`
	DisplayName       string                  `json:"displayName"`
	AvatarURL         string                  `json:"avatarUrl,omitempty"`
	MMR               int                     `json:"mmr"`
	RatingRD          float64                 `json:"ratingRd,omitempty"`
	SeasonID          string                  `json:"seasonId,omitempty"`
	GamesPlayed       int                     `json:"gamesPlayed"`
	Wins              int                     `json:"wins"`
	RankedGamesPlayed int                     `json:"rankedGamesPlayed"`
	RankedWins        int                     `json:"rankedWins"`
	IsGuest           bool                    `json:"isGuest"`
	IsAdmin           bool                    `json:"isAdmin"`
	IsModerator       bool                    `json:"isModerator"`
	IsBanned          bool                    `json:"isBanned"`
	BanReason         string                  `json:"banReason,omitempty"`
	Badges            []contracts.PlayerBadge `json:"badges,omitempty"`
	SelectedBadge     *contracts.PlayerBadge  `json:"selectedBadge,omitempty"`
}

type PublicPlayerProfile struct {
	UserID            string                  `json:"userId"`
	DisplayName       string                  `json:"displayName"`
	AvatarURL         string                  `json:"avatarUrl,omitempty"`
	MMR               int                     `json:"mmr"`
	LeaderboardRank   int                     `json:"leaderboardRank"`
	LeaderboardTotal  int                     `json:"leaderboardTotal"`
	RatingRD          float64                 `json:"ratingRd,omitempty"`
	SeasonID          string                  `json:"seasonId,omitempty"`
	GamesPlayed       int                     `json:"gamesPlayed"`
	Wins              int                     `json:"wins"`
	RankedGamesPlayed int                     `json:"rankedGamesPlayed"`
	RankedWins        int                     `json:"rankedWins"`
	BestWinStreak     int                     `json:"bestWinStreak"`
	PerfectGuesses    int                     `json:"perfectGuesses"`
	FlawlessWins      int                     `json:"flawlessWins"`
	Badges            []contracts.PlayerBadge `json:"badges,omitempty"`
	SelectedBadge     *contracts.PlayerBadge  `json:"selectedBadge,omitempty"`
}
