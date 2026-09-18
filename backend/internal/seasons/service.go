package seasons

import "time"

type Store interface {
	GetRankedSeasonSettings() (RankedSeasonSettings, error)
	SetRankedSeasonResetRule(monthlyResetDay int) (RankedSeasonSettings, error)
	RunDueRankedSeasonReset(now time.Time) (RankedSeasonResetResult, bool, error)
}

type Service struct{ Store }

func NewService(store Store) *Service { return &Service{Store: store} }
