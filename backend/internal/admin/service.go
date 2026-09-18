package admin

type Store interface {
	SearchPlayers(query string, limit int) ([]AdminPlayerSummary, error)
	GetAdminPlayerDetail(userID string) (AdminPlayerDetail, error)
	ListUserRoles() ([]UserRoleGrant, error)
	GrantUserRole(userID, role, grantedBy, reason string) error
	RevokeUserRole(userID, role, revokedBy, reason string) error
}

type Service struct{ Store }

func NewService(store Store) *Service { return &Service{Store: store} }
