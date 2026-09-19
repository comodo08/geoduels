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

// Creator trust tiers. The tier decides quotas and promotion thresholds
// below; values are persisted in gd_users.map_creator_tier.
const (
	mapCreatorTierBase = iota
	mapCreatorTierTrusted
	mapCreatorTierEstablished
)

// mapCreatorSpec is the full definition of a creator tier: upload quotas
// and the promotion thresholds required to enter it. Zero on a promotion
// field means that field is not required.
type mapCreatorSpec struct {
	name                     string
	maxMaps                  int
	maxActiveLocations       int
	maxLocationsPerUpload    int
	maxUploadsPerHour        int
	maxUploadsPerDay         int
	maxUploadedLocationsHour int
	favoritesNeeded          int
	accountAgeDays           int
}

func specForMapCreatorTier(tier int) mapCreatorSpec {
	switch tier {
	case mapCreatorTierEstablished:
		return mapCreatorSpec{
			name:                     "established",
			maxMaps:                  100,
			maxActiveLocations:       100_000_000,
			maxLocationsPerUpload:    1_000_000,
			maxUploadsPerHour:        10,
			maxUploadsPerDay:         30,
			maxUploadedLocationsHour: 5_000_000,
			favoritesNeeded:          50,
			accountAgeDays:           30,
		}
	case mapCreatorTierTrusted:
		return mapCreatorSpec{
			name:                     "trusted",
			maxMaps:                  25,
			maxActiveLocations:       2_500_000,
			maxLocationsPerUpload:    200_000,
			maxUploadsPerHour:        10,
			maxUploadsPerDay:         30,
			maxUploadedLocationsHour: 1_000_000,
			favoritesNeeded:          5,
			accountAgeDays:           14,
		}
	default:
		return mapCreatorSpec{
			name:                     "base",
			maxMaps:                  10,
			maxActiveLocations:       500_000,
			maxLocationsPerUpload:    100_000,
			maxUploadsPerHour:        10,
			maxUploadsPerDay:         30,
			maxUploadedLocationsHour: 300_000,
		}
	}
}

func meetsMapCreatorTier(spec mapCreatorSpec, accountAgeDays, qualifiedFavorites int) bool {
	return accountAgeDays >= spec.accountAgeDays &&
		qualifiedFavorites >= spec.favoritesNeeded
}

func automaticMapCreatorTier(accountAgeDays, qualifiedFavorites int, restricted bool) int {
	if restricted {
		return mapCreatorTierBase
	}
	if meetsMapCreatorTier(specForMapCreatorTier(mapCreatorTierEstablished), accountAgeDays, qualifiedFavorites) {
		return mapCreatorTierEstablished
	}
	if meetsMapCreatorTier(specForMapCreatorTier(mapCreatorTierTrusted), accountAgeDays, qualifiedFavorites) {
		return mapCreatorTierTrusted
	}
	return mapCreatorTierBase
}

func mapCreatorTierName(tier int) string {
	return specForMapCreatorTier(tier).name
}
