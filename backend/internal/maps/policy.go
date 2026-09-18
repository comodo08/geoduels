package maps

// Maps rules that callers and the store must agree on live here so the limits
// are not duplicated between handlers and SQL workflows. Values are carried
// over unchanged from the former persistence implementations.

const (
	// absoluteMaxMapLocations caps the location count of any single map
	// dataset, regardless of the creator's tier.
	absoluteMaxMapLocations = 1_000_000
	// minMapLocations is the smallest playable map; it also relaxes the round
	// requirement for free-for-all and singleplayer matches.
	minMapLocations = 5
	// plannedRoundCount is the number of rounds planned for a standard duel.
	plannedRoundCount = 20
	// mapTrendingWindowDays is the activity window feeding the trending score.
	mapTrendingWindowDays = 7
	// maxMapCommentRunes caps a comment body after whitespace collapsing.
	maxMapCommentRunes = 1000
	// maxMapDisplayNameRunes and maxMapDescriptionRunes bound map metadata.
	maxMapDisplayNameRunes = 80
	maxMapDescriptionRunes = 500
	// maxOfficialRegionCodeRunes bounds an official region code.
	maxOfficialRegionCodeRunes = 32
)

// Creator trust tiers. The tier decides the quota limits below; tier values
// are persisted in gd_users.map_creator_tier.
const (
	mapCreatorTierBase = iota
	mapCreatorTierTrusted
	mapCreatorTierEstablished
)

// Promotion thresholds between tiers.
const (
	trustedFavoritesNeeded     = 25
	trustedAccountAgeDays      = 14
	establishedFavoritesNeeded = 100
	establishedMapsNeeded      = 2
	establishedAccountAgeDays  = 30
)

type mapCreatorLimits struct {
	name                     string
	maxMaps                  int
	maxActiveLocations       int
	maxUploadsPerHour        int
	maxUploadsPerDay         int
	maxUploadedLocationsHour int
}

func limitsForMapCreatorTier(tier int) mapCreatorLimits {
	switch tier {
	case mapCreatorTierEstablished:
		return mapCreatorLimits{
			name:                     "established",
			maxMaps:                  100,
			maxActiveLocations:       1_000_000,
			maxUploadsPerHour:        10,
			maxUploadsPerDay:         30,
			maxUploadedLocationsHour: 1_000_000,
		}
	case mapCreatorTierTrusted:
		return mapCreatorLimits{
			name:                     "trusted",
			maxMaps:                  25,
			maxActiveLocations:       500_000,
			maxUploadsPerHour:        10,
			maxUploadsPerDay:         30,
			maxUploadedLocationsHour: 600_000,
		}
	default:
		return mapCreatorLimits{
			name:                     "base",
			maxMaps:                  10,
			maxActiveLocations:       200_000,
			maxUploadsPerHour:        10,
			maxUploadsPerDay:         30,
			maxUploadedLocationsHour: 300_000,
		}
	}
}

func automaticMapCreatorTier(accountAgeDays, qualifiedFavorites, qualifiedMaps int, restricted bool) int {
	if restricted {
		return mapCreatorTierBase
	}
	if accountAgeDays >= establishedAccountAgeDays && qualifiedFavorites >= establishedFavoritesNeeded && qualifiedMaps >= establishedMapsNeeded {
		return mapCreatorTierEstablished
	}
	if accountAgeDays >= trustedAccountAgeDays && qualifiedFavorites >= trustedFavoritesNeeded {
		return mapCreatorTierTrusted
	}
	return mapCreatorTierBase
}

func mapCreatorTierName(tier int) string {
	return limitsForMapCreatorTier(tier).name
}
