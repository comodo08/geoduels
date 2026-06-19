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
