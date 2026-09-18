package parties

import (
	"os"
	"strings"
	"testing"
)

func TestPartyMapAccessIncludesPublishedCommunityMaps(t *testing.T) {
	body, err := os.ReadFile("../../db/queries/parties.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "published_at is not null") {
		t.Fatal("party map access must include published community maps")
	}
}

func TestPartyReadQueryPreservesNullableMapID(t *testing.T) {
	body, err := os.ReadFile("../../db/queries/parties.sql")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "COALESCE(l.map_id::text, '')") {
		t.Fatal("party read query must preserve nullable uuid map_id")
	}
}
