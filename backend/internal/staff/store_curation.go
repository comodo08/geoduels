package staff

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"geoduels/internal/storekit"
	db "geoduels/pkg/persistence/sqlc/db"
)

func (a *PGStore) stamp(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func (a *PGStore) LockCurationCycle(ctx context.Context) (CurationCycle, error) {
	if _, err := a.requireTx(); err != nil {
		return CurationCycle{}, err
	}
	row, err := a.q().LockMOTWCycle(ctx)
	return CurationCycle{StartsAt: row.StartsAt.Time, ClosesAt: row.ClosesAt.Time}, err
}

func (a *PGStore) CurationWinner(ctx context.Context, start time.Time) (CurationWinner, bool, error) {
	row, err := a.q().MOTWWinner(ctx, a.stamp(start))
	if errors.Is(err, pgx.ErrNoRows) {
		return CurationWinner{}, false, nil
	}
	return CurationWinner{MapID: storekit.UUIDVal(row.MapID), Name: row.DisplayName, CreatorID: storekit.UUIDVal(row.OwnerUserID), Likes: int(row.Likes)}, err == nil, err
}

func (a *PGStore) TrendingCurationMap(ctx context.Context) (CurationWinner, bool, error) {
	row, err := a.q().MOTWTrendingFallback(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return CurationWinner{}, false, nil
	}
	return CurationWinner{MapID: storekit.UUIDVal(row.MapID), Name: row.DisplayName, CreatorID: storekit.UUIDVal(row.OwnerUserID)}, err == nil, err
}

func (a *PGStore) SaveCurationAward(ctx context.Context, start, now time.Time, winner CurationWinner, source string) error {
	return a.q().InsertMOTWAward(ctx, db.InsertMOTWAwardParams{
		CycleStart: a.stamp(start), SelectedAt: a.stamp(now), MapID: storekit.MustUUID(winner.MapID), MapName: winner.Name,
		CreatorUserID: storekit.MustUUID(winner.CreatorID), Source: source, Likes: int32(winner.Likes),
	})
}

func (a *PGStore) AdvanceCurationCycle(ctx context.Context, cycle CurationCycle, awardedAt time.Time) error {
	award := pgtype.Timestamptz{Time: awardedAt, Valid: !awardedAt.IsZero()}
	return a.q().AdvanceMOTWCycle(ctx, db.AdvanceMOTWCycleParams{StartsAt: a.stamp(cycle.StartsAt), ClosesAt: a.stamp(cycle.ClosesAt), CurrentAwardAt: award})
}

func (a *PGStore) NominateMap(ctx context.Context, actor, mapID string, start time.Time) error {
	user, err := storekit.ProfileUUID(actor)
	if err != nil {
		return err
	}
	id, err := storekit.ProfileUUID(mapID)
	if err != nil {
		return ErrUnavailable
	}
	q := a.q()
	currentStart := a.stamp(start)

	_, e := q.NominateMOTWMap(ctx, db.NominateMOTWMapParams{CycleStart: currentStart, ActorID: user, MapID: id})
	if errors.Is(e, pgx.ErrNoRows) {
		return ErrUnavailable
	}
	return e
}

func (a *PGStore) SetNominationLike(ctx context.Context, actor string, id int64, liked bool, start time.Time) error {
	user, err := storekit.ProfileUUID(actor)
	if err != nil {
		return err
	}
	q := a.q()
	currentStart := a.stamp(start)

	exists, e := q.MOTWNominationExists(ctx, db.MOTWNominationExistsParams{ID: id, CycleStart: currentStart})
	if e != nil {
		return e
	}
	if !exists {
		return ErrUnavailable
	}
	if liked {
		return q.SetMOTWLike(ctx, db.SetMOTWLikeParams{NominationID: id, ActorID: user, CycleStart: currentStart})
	}
	return q.RemoveMOTWLike(ctx, db.RemoveMOTWLikeParams{NominationID: id, ActorID: user})
}

func (a *PGStore) ListNominations(ctx context.Context, actor string, page int, cycle CurationCycle) (CurationPage, error) {
	result := CurationPage{Items: []Nomination{}, Page: page, PageSize: 20}
	user, err := storekit.ProfileUUID(actor)
	if err != nil {
		return result, err
	}

	q := a.q()
	currentStart := a.stamp(cycle.StartsAt)

	result.ClosesAt = cycle.ClosesAt
	count, e := q.CountMOTWNominations(ctx, currentStart)
	if e != nil {
		return CurationPage{}, e
	}
	result.Total = int(count)
	rows, e := q.ListMOTWNominations(ctx, db.ListMOTWNominationsParams{CycleStart: currentStart, ActorID: user, PageSize: 20, PageOffset: int32((page - 1) * 20)})
	if e != nil {
		return CurationPage{}, e
	}
	for _, v := range rows {
		result.Items = append(result.Items, Nomination{
			ID: v.ID, MapID: v.MapID.String(), Name: v.DisplayName, ThumbnailKey: storekit.TextVal(v.ThumbnailKey),
			AuthorName: v.AuthorName, Likes: int(v.Likes), Liked: v.Liked, NominatedAt: v.NominatedAt.Time,
		})
	}
	return result, nil
}
