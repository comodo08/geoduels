package persistence

import "testing"

func TestSeasonRankBadgeFromParts(t *testing.T) {
	tests := []struct {
		name      string
		seasonID  string
		rank      int
		label     string
		imageURL  string
		owned     bool
		descMatch string
	}{
		{
			name:      "first place",
			seasonID:  "s3",
			rank:      1,
			label:     "Season 3 Champion",
			imageURL:  "/medals/season-1st-medal.v1.png",
			owned:     true,
			descMatch: "Finished #1 in Season 3.",
		},
		{
			name:      "second place",
			seasonID:  "s3",
			rank:      2,
			label:     "Season 3 Runner-Up",
			imageURL:  "/medals/season-2nd-medal.v1.png",
			owned:     true,
			descMatch: "Finished #2 in Season 3.",
		},
		{
			name:      "third place",
			seasonID:  "s3",
			rank:      3,
			label:     "Season 3 Third Place",
			imageURL:  "/medals/season-3rd-medal.v1.png",
			owned:     true,
			descMatch: "Finished #3 in Season 3.",
		},
		{
			name:      "other top 100",
			seasonID:  "s3",
			rank:      57,
			label:     "Season 3 #57",
			imageURL:  "/medals/platinum-medal.v1.png",
			owned:     true,
			descMatch: "Finished #57 in Season 3.",
		},
		{
			name:      "hundredth",
			seasonID:  "s3",
			rank:      100,
			label:     "Season 3 #100",
			imageURL:  "/medals/platinum-medal.v1.png",
			owned:     true,
			descMatch: "Finished #100 in Season 3.",
		},
		{
			name:      "unowned template",
			seasonID:  "s3",
			rank:      0,
			label:     "Season 3 Top 100",
			imageURL:  "/medals/platinum-medal.v1.png",
			owned:     false,
			descMatch: "Awarded to players who finish in the top 100 when Season 3 ends.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			badge, ok := badgeFromParts(badgeCodeSeasonRank, tt.seasonID, tt.rank, tt.owned)
			if !ok {
				t.Fatal("badgeFromParts returned not ok")
			}
			if badge.ID != seasonRankBadgeID(tt.seasonID) {
				t.Fatalf("badge id = %q, want %q", badge.ID, seasonRankBadgeID(tt.seasonID))
			}
			if badge.Label != tt.label {
				t.Fatalf("label = %q, want %q", badge.Label, tt.label)
			}
			if badge.ImageURL != tt.imageURL {
				t.Fatalf("image url = %q, want %q", badge.ImageURL, tt.imageURL)
			}
			if badge.Rank != tt.rank {
				t.Fatalf("rank = %d, want %d", badge.Rank, tt.rank)
			}
			if badge.Owned != tt.owned {
				t.Fatalf("owned = %v, want %v", badge.Owned, tt.owned)
			}
			if badge.Kind != "season_rank" {
				t.Fatalf("kind = %q, want season_rank", badge.Kind)
			}
			if badge.Description == "" || badge.Description != tt.descMatch {
				t.Fatalf("description = %q, want %q", badge.Description, tt.descMatch)
			}
		})
	}
}

func TestSeasonRankBadgeIDRoundTrip(t *testing.T) {
	for _, seasonID := range []string{"s2", "s2.5", "s3"} {
		id := seasonRankBadgeID(seasonID)
		ref, ok := badgeRefFromID(id)
		if !ok {
			t.Fatalf("badgeRefFromID(%q) not ok", id)
		}
		if ref.Code != badgeCodeSeasonRank {
			t.Fatalf("badgeRefFromID(%q) code = %d, want %d", id, ref.Code, badgeCodeSeasonRank)
		}
		if ref.SeasonID != seasonID {
			t.Fatalf("badgeRefFromID(%q) season = %q, want %q", id, ref.SeasonID, seasonID)
		}
		if got := badgeIDFromParts(ref.Code, ref.SeasonID); got != id {
			t.Fatalf("badgeIDFromParts(%d, %q) = %q, want %q", ref.Code, ref.SeasonID, got, id)
		}
	}
}
