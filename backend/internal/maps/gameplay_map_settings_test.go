package maps

import (
	"testing"

	"geoduels/pkg/contracts"
)

func TestDecodeGameplayMapSettingsMigratesLegacyKeys(t *testing.T) {
	t.Parallel()
	settings := decodeGameplayMapSettings(`{
		"rankedMovingMapId":"legacy-moving",
		"rankedNmpzMapId":"legacy-ranked-nmpz",
		"singleplayerMovingMapId":"legacy-sp-moving",
		"singleplayerNmpzMapId":"legacy-sp-nmpz"
	}`)
	if settings.MovingMapID != "legacy-moving" {
		t.Fatalf("MovingMapID = %q, want legacy-moving", settings.MovingMapID)
	}
	if settings.NoMoveMapID != "legacy-ranked-nmpz" {
		t.Fatalf("NoMoveMapID = %q, want legacy-ranked-nmpz", settings.NoMoveMapID)
	}
	if settings.NMPZMapID != "legacy-sp-nmpz" {
		t.Fatalf("NMPZMapID = %q, want legacy-sp-nmpz", settings.NMPZMapID)
	}
}

func TestDecodeGameplayMapSettingsPrefersModeKeys(t *testing.T) {
	t.Parallel()
	settings := decodeGameplayMapSettings(`{
		"movingMapId":"mode-moving",
		"noMoveMapId":"mode-no-move",
		"nmpzMapId":"mode-nmpz",
		"rankedMovingMapId":"legacy-moving",
		"rankedNmpzMapId":"legacy-ranked-nmpz",
		"singleplayerNmpzMapId":"legacy-sp-nmpz"
	}`)
	want := contracts.GameplayMapSettings{
		MovingMapID: "mode-moving",
		NoMoveMapID: "mode-no-move",
		NMPZMapID:   "mode-nmpz",
	}
	if settings != want {
		t.Fatalf("settings = %#v, want %#v", settings, want)
	}
}
