package profiles

type Store interface {
	UpsertUser(userID, email, displayName string) error
	GetProfile(userID string) (Profile, error)
	GetPublicPlayerProfileByNickname(nickname string) (PublicPlayerProfile, error)
	UpdateSelectedBadge(userID, badgeID string) (Profile, error)
}

type Service struct{ Store }

func NewService(store Store) *Service { return &Service{Store: store} }
