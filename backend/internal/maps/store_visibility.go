package maps

import (
	"strconv"
	"strings"
)

// communityMapListPredicate is the SQL visibility contract for listings that
// surface community maps: the map must have an owner, currently be public,
// and be ready to play. Historical publication state never qualifies.
const communityMapListPredicate = "m.owner_user_id is not null and m.visibility='public' and m.status='ready'"

// mapVisibleToUserSQL builds the visibility predicate for a map row alias.
func mapVisibleToUserSQL(alias string, userArg int, includeUnlisted bool) string {
	qualifier := strings.TrimSpace(alias)
	if qualifier != "" {
		qualifier += "."
	}
	predicate := "(" +
		qualifier + "owner_user_id is null" +
		" or " + qualifier + "official_at is not null" +
		" or " + qualifier + "owner_user_id = nullif($" + strconv.Itoa(userArg) + ",'')::uuid" +
		" or " + qualifier + "visibility = 'public'"
	if includeUnlisted {
		predicate += " or " + qualifier + "visibility = 'unlisted'"
	}
	return predicate + ")"
}

func mapVisibleToUser(ownerUserID, accessUserID, visibility string, official bool) bool {
	if ownerUserID == "" || official || ownerUserID == accessUserID {
		return true
	}
	switch strings.TrimSpace(strings.ToLower(visibility)) {
	case "public", "unlisted":
		return true
	default:
		return false
	}
}
