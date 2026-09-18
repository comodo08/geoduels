package badges

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"geoduels/internal/storekit"
)

func profileUUID(v string) (pgtype.UUID, error)  { return storekit.ProfileUUID(v) }
func chatUUID(s string) pgtype.UUID              { return storekit.MustUUID(s) }
func timestamptz(t time.Time) pgtype.Timestamptz { return storekit.Timestamptz(t) }
