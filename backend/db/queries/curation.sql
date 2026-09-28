-- name: LockMOTWCycle :one
SELECT starts_at,closes_at FROM motw_schedule WHERE singleton=true FOR UPDATE;

-- name: NominateMOTWMap :one
INSERT INTO motw_nominations(cycle_start,map_id,nominated_by)
SELECT sqlc.arg(cycle_start),m.id,sqlc.arg(actor_id) FROM maps m
WHERE m.id=sqlc.arg(map_id) AND status='ready' AND visibility='public' AND archived_at IS NULL AND owner_user_id IS NOT NULL
ON CONFLICT(cycle_start,map_id) DO UPDATE SET map_id=excluded.map_id RETURNING id;

-- name: SetMOTWLike :exec
INSERT INTO motw_likes(nomination_id,user_id)
SELECT id,sqlc.arg(actor_id) FROM motw_nominations WHERE id=sqlc.arg(nomination_id) AND cycle_start=sqlc.arg(cycle_start)
ON CONFLICT DO NOTHING;

-- name: RemoveMOTWLike :exec
DELETE FROM motw_likes WHERE nomination_id=sqlc.arg(nomination_id) AND user_id=sqlc.arg(actor_id);

-- name: MOTWNominationExists :one
SELECT EXISTS(SELECT 1 FROM motw_nominations WHERE id=$1 AND cycle_start=$2);

-- name: ListMOTWNominations :many
SELECT n.id,n.map_id,m.display_name,m.thumbnail_key,coalesce(u.display_name,'GeoDuels') AS author_name,n.nominated_at,
 (SELECT count(*) FROM motw_likes l WHERE l.nomination_id=n.id)::int AS likes,
 EXISTS(SELECT 1 FROM motw_likes l WHERE l.nomination_id=n.id AND l.user_id=sqlc.arg(actor_id)) AS liked
FROM motw_nominations n JOIN maps m ON m.id=n.map_id LEFT JOIN users u ON u.id=m.owner_user_id
WHERE n.cycle_start=sqlc.arg(cycle_start) AND m.status='ready' AND m.visibility='public' AND m.archived_at IS NULL AND m.owner_user_id IS NOT NULL
ORDER BY likes DESC,n.nominated_at,n.id LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountMOTWNominations :one
SELECT count(*)::int FROM motw_nominations n JOIN maps m ON m.id=n.map_id
WHERE n.cycle_start=$1 AND m.status='ready' AND m.visibility='public' AND m.archived_at IS NULL AND m.owner_user_id IS NOT NULL;

-- name: MOTWWinner :one
SELECT n.map_id,m.display_name,m.owner_user_id,(SELECT count(*) FROM motw_likes l WHERE l.nomination_id=n.id)::int AS likes
FROM motw_nominations n JOIN maps m ON m.id=n.map_id
WHERE n.cycle_start=$1 AND m.status='ready' AND m.visibility='public' AND m.archived_at IS NULL AND m.owner_user_id IS NOT NULL
ORDER BY likes DESC,n.nominated_at,n.id LIMIT 1;

-- name: MOTWTrendingFallback :one
SELECT m.id AS map_id,m.display_name,m.owner_user_id
FROM maps m LEFT JOIN LATERAL (
 SELECT coalesce(sum(unique_players)*3+sum(unique_favoriters)*8+sum(unique_commenters)*2,0)::float8 AS weighted,
 coalesce(sum(unique_players)+sum(unique_favoriters)+sum(unique_commenters),0)::float8 AS activity
 FROM map_stats_daily d WHERE d.map_id=m.id AND d.day >= current_date-7
) score ON true
WHERE m.status='ready' AND m.visibility='public' AND m.archived_at IS NULL AND m.owner_user_id IS NOT NULL
ORDER BY (score.weighted * CASE WHEN score.activity>=3 AND m.published_at IS NOT NULL THEN
 1+19*(1-least(greatest(extract(epoch FROM (now()-m.published_at))/604800,0),1)) ELSE 1 END) DESC,
 m.published_at DESC NULLS LAST,m.id LIMIT 1;

-- name: InsertMOTWAward :exec
INSERT INTO motw_awards(cycle_start,selected_at,map_id,map_name,creator_user_id,source,likes)
VALUES(sqlc.arg(cycle_start),sqlc.arg(selected_at),sqlc.arg(map_id),sqlc.arg(map_name),sqlc.arg(creator_user_id),sqlc.arg(source),sqlc.arg(likes));

-- name: AdvanceMOTWCycle :exec
UPDATE motw_schedule SET starts_at=sqlc.arg(starts_at),closes_at=sqlc.arg(closes_at),current_award_at=coalesce(sqlc.narg(current_award_at),current_award_at) WHERE singleton=true;
