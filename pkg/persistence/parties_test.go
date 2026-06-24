package persistence

import (
	"strings"
	"testing"
)

func TestPartyMapAccessIncludesPublishedCommunityMaps(t *testing.T) {
	if !strings.Contains(partyMapAccessiblePredicate, "published_at is not null") {
		t.Fatal("party map access must include published community maps")
	}
}

func TestPartyReadQueryCastsNullableMapID(t *testing.T) {
	query := partyReadQuery("l.id = $1")
	if !strings.Contains(query, "coalesce(l.map_id::text, '')") {
		t.Fatal("party read query must cast nullable uuid map_id before coalescing with text")
	}
}
