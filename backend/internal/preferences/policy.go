package preferences

import "errors"

// SupportedSchemaVersion is the only preferences payload version accepted.
const SupportedSchemaVersion = 1

var (
	ErrRevisionConflict   = errors.New("preference revision conflict")
	ErrUnsupportedVersion = errors.New("unsupported preference version")
)

// ValidateVersion is the write gate for preference payloads.
func ValidateVersion(version int) error {
	if version != SupportedSchemaVersion {
		return ErrUnsupportedVersion
	}
	return nil
}
