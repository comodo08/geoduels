package maps

// Package maps owns the custom-map domain: the public catalog, uploads and
// revisions, comments, creator trust tiers, and official map administration.
// Map DTOs exchanged with clients (CustomMap, MapDetails, MapListOptions,
// MapUploadQuota, MapComment, MapCommentCreate, CustomMapUpdate,
// MapImportSummary) live in pkg/contracts and are referenced directly.

// OfficialMapImportInput describes an administrator-driven import of an
// official map keyed by its canonical map key.
type OfficialMapImportInput struct {
	MapKey             string
	DisplayName        string
	Description        string
	Visibility         string
	Difficulty         string
	ThumbnailKey       string
	ThumbnailVariant   int
	OfficialRegionType string
	OfficialRegionCode string
}
