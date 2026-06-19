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
	"math"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultMapKey = "a-source-world"
const defaultPostgresURL = "postgres://geoduels:geoduels@127.0.0.1:5432/geoduels?sslmode=disable"

type row struct {
	LatE7       int32
	LngE7       int32
	Country     string
	PanoID      *string
	HeadingCDeg *int16
	PitchCDeg   *int16
	RandKey     int32
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
			storage_id integer generated always as identity unique,
			map_key text not null references maps(map_key) on delete cascade,
			content_hash text not null,
			status text not null default 'validated',
			row_count integer not null default 0,
			created_at timestamptz not null default now(),
			unique(map_key, content_hash)
		);
		create table if not exists locations (
			revision_storage_id integer not null references map_revisions(storage_id) on delete cascade,
			lat_e7 integer not null,
			lng_e7 integer not null,
			rand_key_i integer not null,
			heading_cdeg smallint,
			pitch_cdeg smallint,
			country text,
			pano_id text
		);
		alter table map_revisions add column if not exists storage_id integer generated always as identity;
		create unique index if not exists map_revisions_storage_id_key on map_revisions(storage_id);
		alter table locations add column if not exists revision_storage_id integer references map_revisions(storage_id) on delete cascade;
		create index if not exists idx_locations_revision_rand on locations(revision_storage_id, rand_key_i);
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
		if err := pool.QueryRow(ctx, `select count(*) from locations where revision_storage_id=(select storage_id from map_revisions where id=$1)`, existing).Scan(&count); err != nil {
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
	var revisionStorageID int32
	if err := tx.QueryRow(ctx, `select storage_id from map_revisions where id=$1`, revisionID).Scan(&revisionStorageID); err != nil {
		return err
	}

	batchSize := 2000
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		block := make([][]any, 0, end-i)
		for _, r := range rows[i:end] {
			block = append(block, []any{
				revisionStorageID,
				r.LatE7,
				r.LngE7,
				r.Country,
				r.PanoID,
				r.HeadingCDeg,
				r.PitchCDeg,
				r.RandKey,
			})
		}
		if _, err := tx.CopyFrom(
			ctx,
			pgx.Identifier{"locations"},
			[]string{"revision_storage_id", "lat_e7", "lng_e7", "country", "pano_id", "heading_cdeg", "pitch_cdeg", "rand_key_i"},
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
		select $1, coalesce(nullif(country,''), 'Unknown'), count(*)::int
		from locations
		where revision_storage_id=$2
		group by coalesce(nullif(country,''), 'Unknown')
	`, revisionID, revisionStorageID); err != nil {
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
		r := row{
			LatE7:   int32(math.Round(lat * 10_000_000)),
			LngE7:   int32(math.Round(lng * 10_000_000)),
			RandKey: stableRand(lat, lng),
		}
		if c, ok := it["country"].(string); ok {
			r.Country = c
		}
		if pano, ok := it["panoId"].(string); ok && pano != "" {
			r.PanoID = &pano
		}
		if h, ok := asFloat(it["heading"]); ok {
			value := compactAngle(h, false)
			r.HeadingCDeg = &value
		}
		if p, ok := asFloat(it["pitch"]); ok {
			value := compactAngle(p, true)
			r.PitchCDeg = &value
		}
		out = append(out, r)
	}
	return out, nil
}

func stableRand(lat, lng float64) int32 {
	h := sha1.Sum([]byte(fmt.Sprintf("%.8f:%.8f", lat, lng)))
	v := int(h[0])<<16 | int(h[1])<<8 | int(h[2])
	return int32(v)
}

func compactAngle(value float64, pitch bool) int16 {
	if pitch {
		value = math.Max(-90, math.Min(90, value))
	} else {
		value = math.Mod(value+180, 360)
		if value < 0 {
			value += 360
		}
		value -= 180
	}
	return int16(math.Round(value * 100))
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
