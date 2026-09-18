package maps

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"geoduels/pkg/contracts"
	db "geoduels/pkg/persistence/sqlc/db"
)

// Gameplay map configuration: which map answers each mode/ruleset slot.
// The maps feature owns catalog CRUD; this stays with persistence because
// match coordination, parties, and match planning resolve maps at runtime.

const gameplayMapSettingsKey = "gameplay_map_settings"

func defaultGameplayMapSettings() contracts.GameplayMapSettings {
	return contracts.GameplayMapSettings{
		MovingMapID: contracts.MapKeyMoving,
		NoMoveMapID: contracts.MapKeyNMPZ,
		NMPZMapID:   contracts.MapKeyNMPZ,
	}
}

type legacyGameplayMapSettings struct {
	MovingMapID             string `json:"movingMapId"`
	NoMoveMapID             string `json:"noMoveMapId"`
	NMPZMapID               string `json:"nmpzMapId"`
	RankedMovingMapID       string `json:"rankedMovingMapId"`
	RankedNMPZMapID         string `json:"rankedNmpzMapId"`
	SingleplayerMovingMapID string `json:"singleplayerMovingMapId"`
	SingleplayerNMPZMapID   string `json:"singleplayerNmpzMapId"`
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeGameplayMapSettings(settings contracts.GameplayMapSettings) contracts.GameplayMapSettings {
	defaults := defaultGameplayMapSettings()
	if strings.TrimSpace(settings.MovingMapID) == "" {
		settings.MovingMapID = defaults.MovingMapID
	}
	if strings.TrimSpace(settings.NoMoveMapID) == "" {
		settings.NoMoveMapID = defaults.NoMoveMapID
	}
	if strings.TrimSpace(settings.NMPZMapID) == "" {
		settings.NMPZMapID = defaults.NMPZMapID
	}
	return settings
}

func decodeGameplayMapSettings(raw string) contracts.GameplayMapSettings {
	var legacy legacyGameplayMapSettings
	if err := json.Unmarshal([]byte(raw), &legacy); err != nil {
		return defaultGameplayMapSettings()
	}
	return normalizeGameplayMapSettings(contracts.GameplayMapSettings{
		MovingMapID: firstNonEmpty(legacy.MovingMapID, legacy.RankedMovingMapID, legacy.SingleplayerMovingMapID),
		// Ranked NMPZ became ranked No Move; keep that map for the no-move slot.
		NoMoveMapID: firstNonEmpty(legacy.NoMoveMapID, legacy.RankedNMPZMapID),
		NMPZMapID:   firstNonEmpty(legacy.NMPZMapID, legacy.SingleplayerNMPZMapID, legacy.RankedNMPZMapID),
	})
}

func (s *PGStore) GetGameplayMapSettings() (contracts.GameplayMapSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	return s.gameplayMapSettings(ctx, s.pool)
}

func (s *PGStore) gameplayMapSettings(ctx context.Context, q db.DBTX) (contracts.GameplayMapSettings, error) {
	raw, err := db.New(q).GetGameplayMapSettings(ctx, gameplayMapSettingsKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return defaultGameplayMapSettings(), nil
		}
		return contracts.GameplayMapSettings{}, err
	}
	return decodeGameplayMapSettings(string(raw)), nil
}

func (s *PGStore) ResolveGameplayMapID(mode contracts.MatchMode, ruleset contracts.GameRuleset, requestedMapID string) (string, error) {
	requestedMapID = strings.TrimSpace(requestedMapID)
	if requestedMapID != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()
		id, _, err := resolveMapIdentity(ctx, s.pool, requestedMapID)
		return id, err
	}
	settings, err := s.GetGameplayMapSettings()
	if err != nil {
		return "", err
	}
	switch contracts.NormalizeRuleset(ruleset) {
	case contracts.RulesetNoMove:
		return s.ResolveGameplayMapID(mode, ruleset, settings.NoMoveMapID)
	case contracts.RulesetNMPZ:
		return s.ResolveGameplayMapID(mode, ruleset, settings.NMPZMapID)
	default:
		return s.ResolveGameplayMapID(mode, ruleset, settings.MovingMapID)
	}
}
