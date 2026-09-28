package staff

import (
	"context"
	"time"
)

type CurationCycle struct{ StartsAt, ClosesAt time.Time }
type CurationWinner struct {
	MapID, Name, CreatorID string
	Likes                  int
}

// NextCurationCycle advances past downtime without inventing awards for unplayed weeks.
func NextCurationCycle(closesAt, now time.Time) (time.Time, time.Time) {
	start := closesAt
	for !start.Add(CurationWindow).After(now) {
		start = start.Add(CurationWindow)
	}
	return start, start.Add(CurationWindow)
}

// withCurationCycle serializes selection, awards and the caller's nomination
// operation in the same transaction. Candidate ranking stays a store read;
// choosing the fallback and awarding the creator are application decisions.
func (s *Service) withCurationCycle(ctx context.Context, fn func(Store, CurationCycle) error) error {
	return s.store.WithinTx(ctx, func(store Store) error {
		cycle, err := store.LockCurationCycle(ctx)
		if err != nil {
			return err
		}
		now := s.now()
		if !now.Before(cycle.ClosesAt) {
			winner, found, err := store.CurationWinner(ctx, cycle.StartsAt)
			if err != nil {
				return err
			}
			source := "nomination"
			if !found {
				winner, found, err = store.TrendingCurationMap(ctx)
				if err != nil {
					return err
				}
				source = "trending"
			}
			var awardedAt time.Time
			if found {
				if err := store.SaveCurationAward(ctx, cycle.StartsAt, now, winner, source); err != nil {
					return err
				}
				if winner.CreatorID != "" {
					if _, err := store.AwardBadge(ctx, winner.CreatorID, "map-of-the-week"); err != nil {
						return err
					}
				}
				awardedAt = cycle.StartsAt
			}
			cycle.StartsAt, cycle.ClosesAt = NextCurationCycle(cycle.ClosesAt, now)
			if err := store.AdvanceCurationCycle(ctx, cycle, awardedAt); err != nil {
				return err
			}
		}
		return fn(store, cycle)
	})
}
