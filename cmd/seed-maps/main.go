package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"geoduels/pkg/persistence"
)

// seedDefaultMaps loads the bundled sample map source into the database as the
// official dev default maps. The map keys match the gameplay_map_settings rows
// (a-source-world for moving modes, a-location-world for NMPZ modes) so the game
// is immediately playable after a fresh clone + migrate.
func main() {
	source := flag.String("source", "datasets/a-source-world.sample.json", "path to the map source JSON")
	flag.Parse()

	if os.Getenv("POSTGRES_URL") == "" {
		// When run on the host (not inside a container), talk to the published
		// Postgres port. POSTGRES_URL can be set to override this.
		os.Setenv("POSTGRES_URL", "postgres://geoduels:geoduels@localhost:5432/geoduels?sslmode=disable")
	}

	store, err := persistence.NewFromEnv()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer store.Close()

	cat, ok := store.(persistence.MapCatalog)
	if !ok {
		log.Fatal("store does not implement persistence.MapCatalog")
	}

	f, err := os.Open(*source)
	if err != nil {
		log.Fatalf("failed to open source file %q: %v", *source, err)
	}
	defer f.Close()

	targets := []struct {
		key  string
		name string
	}{
		{"a-source-world", "World (Sample)"},
		{"a-location-world", "World Locations (Sample)"},
	}

	for _, t := range targets {
		if _, err := f.Seek(0, 0); err != nil {
			log.Fatalf("failed to rewind source: %v", err)
		}
		input := persistence.OfficialMapImportInput{
			MapKey:           t.key,
			DisplayName:      t.name,
			Description:      "Bundled sample world map for local development.",
			Visibility:       "public",
			Difficulty:       "normal",
			ThumbnailVariant: 1,
		}
		m, err := cat.ImportOfficialMap("", input, f)
		if err != nil {
			log.Fatalf("failed to seed map %q: %v", t.key, err)
		}
		fmt.Printf("seeded map %q (%s) -> %d locations\n", m.DisplayName, m.MapKey, m.LocationCount)
	}
}
