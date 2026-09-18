package maps

import "testing"

func TestGameplayMapRoleField(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"moving":              "movingMapId",
		"no_move":             "noMoveMapId",
		"nmpz":                "nmpzMapId",
		"ranked_moving":       "movingMapId",
		"ranked_nmpz":         "noMoveMapId",
		"singleplayer_moving": "movingMapId",
		"singleplayer_nmpz":   "nmpzMapId",
	}
	for role, want := range cases {
		got, err := gameplayMapRoleField(role)
		if err != nil {
			t.Fatalf("role %q: %v", role, err)
		}
		if got != want {
			t.Fatalf("role %q = %q, want %q", role, got, want)
		}
	}
}
