package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"

	"geoduels/internal/maps"
	"geoduels/pkg/contracts"
	"geoduels/pkg/persistence"
)

// mapStore resolves configured gameplay maps (persistence-owned settings);
// the maps feature store owns dataset imports.
type mapStore interface {
	ResolveGameplayMapID(mode contracts.MatchMode, ruleset contracts.GameRuleset, requestedMapID string) (string, error)
}

type mapDatasetStore interface {
	ReplaceMapLocations(mapKey, displayName string, dataset []byte) (contracts.MapImportSummary, error)
}

type requiredMap struct {
	key         string
	displayName string
	mode        contracts.MatchMode
	ruleset     contracts.GameRuleset
}

var requiredDevelopmentMaps = []requiredMap{
	{key: contracts.MapKeyMoving, displayName: "Sample World", mode: contracts.ModeDuel, ruleset: contracts.RulesetMoving},
	{key: contracts.MapKeyNMPZ, displayName: "Sample Varied World", mode: contracts.ModeDuel, ruleset: contracts.RulesetNoMove},
	{key: contracts.MapKeyNMPZ, displayName: "Sample Varied World", mode: contracts.ModeSingleplayer, ruleset: contracts.RulesetNMPZ},
}

func main() {
	datasetPath := flag.String("dataset", "../maps/datasets/a-source-world.sample.json", "sample map dataset")
	flag.Parse()

	dataset, err := os.ReadFile(*datasetPath)
	if err != nil {
		log.Fatalf("read sample map dataset: %v", err)
	}
	store, err := persistence.NewFromEnv()
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer store.Close()
	datasets := maps.NewPGStore(store.Pool())

	for _, required := range requiredDevelopmentMaps {
		created, err := ensureDevelopmentMap(datasets, datasets, required, dataset)
		if err != nil {
			log.Fatalf("bootstrap %s: %v", required.key, err)
		}
		if created {
			fmt.Printf("created development map %s from %s\n", required.key, *datasetPath)
		} else {
			fmt.Printf("development map %s already configured; skipped\n", required.key)
		}
	}
}

func ensureDevelopmentMap(store mapStore, datasets mapDatasetStore, required requiredMap, dataset []byte) (bool, error) {
	if _, err := store.ResolveGameplayMapID(required.mode, required.ruleset, ""); err == nil {
		return false, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return false, fmt.Errorf("check configured map: %w", err)
	}
	if _, err := datasets.ReplaceMapLocations(required.key, required.displayName, dataset); err != nil {
		return false, fmt.Errorf("import sample dataset: %w", err)
	}
	return true, nil
}
