package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultMapKey = "a-source-world"
const defaultPostgresURL = "postgres://geoduels:geoduels@127.0.0.1:5432/geoduels?sslmode=disable"

type row struct {
	Lat     float64
	Lng     float64
	Country string
	PanoID  *string
	Heading *float64
	Pitch   *float64
	RandKey float64
}

func main() {
	dataset := flag.String("dataset", "datasets/a-source-world.json", "dataset file")
	mapKey := flag.String("map-key", defaultMapKey, "map key")
	timeout := flag.Duration("timeout", 30*time.Minute, "overall DB ingest timeout")
	flag.Parse()

	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL == "" {
		dbURL = defaultPostgresURL
	}

	b, err := os.ReadFile(*dataset)
	if err != nil {
		log.Fatal(err)
	}
	rows, err := parseRows(b)
	if err != nil {
		log.Fatal(err)
	}
	if len(rows) == 0 {
		log.Fatal("no valid rows")
	}

	h := sha1.Sum(b)
	sourceHash := hex.EncodeToString(h[:])

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	connectCtx, connectCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer connectCancel()
	pool, err := pgxpool.New(connectCtx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(connectCtx); err != nil {
		log.Fatal(err)
	}

	if err := ensureSchema(ctx, pool); err != nil {
		log.Fatal(err)
	}
	revisionID, inserted, err := upsertRevision(ctx, pool, *mapKey, sourceHash)
	if err != nil {
		log.Fatal(err)
	}
	if inserted {
		if err := ingestRows(ctx, pool, revisionID, rows); err != nil {
			log.Fatal(err)
		}
	}
	if err := activateRevision(ctx, pool, *mapKey, revisionID); err != nil {
		log.Fatal(err)
	}
	log.Printf("location ingest complete map=%s revision=%s inserted=%t rows=%d", *mapKey, revisionID, inserted, len(rows))
}

func ensureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		create table if not exists maps (
			map_key text primary key,
			display_name text not null,
			created_at timestamptz not null default now()
		);
		create table if not exists map_revisions (
			id text primary key,
			map_key text not null references maps(map_key) on delete cascade,
			content_hash text not null,
			status text not null default 'validated',
			row_count integer not null default 0,
			created_at timestamptz not null default now(),
			unique(map_key, content_hash)
		);
		create table if not exists locations (
			id bigserial primary key,
			map_revision_id text references map_revisions(id) on delete cascade,
			lat double precision not null,
			lng double precision not null,
			country text,
			pano_id text,
			heading double precision,
			pitch double precision,
			rand_key double precision not null
		);
		alter table locations add column if not exists map_revision_id text references map_revisions(id) on delete cascade;
		create index if not exists idx_locations_revision_rand on locations(map_revision_id, rand_key);
		create index if not exists idx_locations_revision_id on locations(map_revision_id, id);
	`)
	return err
}

func upsertRevision(ctx context.Context, pool *pgxpool.Pool, mapKey, sourceHash string) (revisionID string, shouldIngest bool, err error) {
	revisionID = mapKey + "-" + sourceHash[:12]
	if _, err := pool.Exec(ctx, `
		insert into maps(map_key, display_name) values($1, $2)
		on conflict (map_key) do nothing
	`, mapKey, mapKey); err != nil {
		return "", false, err
	}
	var existing string
	err = pool.QueryRow(ctx, `select id from map_revisions where map_key=$1 and content_hash=$2 limit 1`, mapKey, sourceHash).Scan(&existing)
	if err == nil {
		var count int64
		if err := pool.QueryRow(ctx, `select count(*) from locations where map_revision_id=$1`, existing).Scan(&count); err != nil {
			return "", false, err
		}
		return existing, count == 0, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", false, err
	}
	_, err = pool.Exec(ctx, `
		insert into map_revisions(id, map_key, content_hash, status)
		values($1, $2, $3, 'validated')
	`, revisionID, mapKey, sourceHash)
	if err != nil {
		return "", false, err
	}
	return revisionID, true, nil
}

func ingestRows(ctx context.Context, pool *pgxpool.Pool, revisionID string, rows []row) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	batchSize := 2000
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		block := make([][]any, 0, end-i)
		for _, r := range rows[i:end] {
			block = append(block, []any{
				revisionID,
				r.Lat,
				r.Lng,
				r.Country,
				r.PanoID,
				r.Heading,
				r.Pitch,
				r.RandKey,
			})
		}
		if _, err := tx.CopyFrom(
			ctx,
			pgx.Identifier{"locations"},
			[]string{"map_revision_id", "lat", "lng", "country", "pano_id", "heading", "pitch", "rand_key"},
			pgx.CopyFromRows(block),
		); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `update map_revisions set row_count=$2 where id=$1`, revisionID, len(rows)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `delete from map_revision_country_stats where map_revision_id=$1`, revisionID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		insert into map_revision_country_stats(map_revision_id, country, location_count)
		select map_revision_id, coalesce(nullif(country,''), 'Unknown'), count(*)::int
		from locations
		where map_revision_id=$1
		group by map_revision_id, coalesce(nullif(country,''), 'Unknown')
	`, revisionID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func activateRevision(ctx context.Context, pool *pgxpool.Pool, mapKey, revisionID string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `update map_revisions set status='active' where id=$1`, revisionID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		update maps
		set active_revision_id=$2,
			status='ready',
			location_count=(select row_count from map_revisions where id=$2),
			updated_at=now()
		where map_key=$1
	`, mapKey, revisionID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func parseRows(b []byte) ([]row, error) {
	var raw []map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	out := make([]row, 0, len(raw))
	for _, it := range raw {
		lat, ok1 := asFloat(it["lat"])
		lng, ok2 := asFloat(it["lng"])
		if !ok1 || !ok2 {
			continue
		}
		if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
			continue
		}
		r := row{Lat: lat, Lng: lng, RandKey: stableRand(lat, lng)}
		if c, ok := it["country"].(string); ok {
			r.Country = c
		}
		if pano, ok := it["panoId"].(string); ok && pano != "" {
			r.PanoID = &pano
		}
		if h, ok := asFloat(it["heading"]); ok {
			r.Heading = &h
		}
		if p, ok := asFloat(it["pitch"]); ok {
			r.Pitch = &p
		}
		out = append(out, r)
	}
	return out, nil
}

func stableRand(lat, lng float64) float64 {
	h := sha1.Sum([]byte(fmt.Sprintf("%.8f:%.8f", lat, lng)))
	v := int(h[0])<<16 | int(h[1])<<8 | int(h[2])
	return float64(v) / float64(1<<24)
}

func asFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	default:
		return 0, false
	}
}
