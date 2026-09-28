package parties

import "errors"

// ErrNotFound identifies a missing party or membership.
var ErrNotFound = errors.New("no rows in result set")
