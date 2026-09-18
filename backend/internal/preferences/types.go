package preferences

import "encoding/json"

type UserPreferences struct {
	SchemaVersion int
	Preferences   json.RawMessage
	Revision      int64
}
