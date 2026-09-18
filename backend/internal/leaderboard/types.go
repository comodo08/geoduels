package leaderboard

type Entry struct {
	Rank        int    `json:"rank"`
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
	MMR         int    `json:"mmr"`
	GamesPlayed int    `json:"gamesPlayed"`
	Wins        int    `json:"wins"`
}

type Overview struct {
	Mode                   string
	SeasonID               string
	SelfRank, TotalPlayers int
	Entries                []Entry
}
