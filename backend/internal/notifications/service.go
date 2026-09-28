package notifications

import (
	"context"

	"geoduels/pkg/contracts"
)

type Store interface {
	ListUserNotifications(string, int) ([]contracts.UserNotification, error)
	ListNotificationInbox(string, int, int64) ([]contracts.UserNotification, error)
	MarkUserNotificationRead(string, int64) error
	MarkAllUserNotificationsRead(string) error
}

type Service struct{ store Store }

func NewService(store Store) *Service { return &Service{store: store} }
func (s *Service) List(ctx context.Context, user string, limit int) ([]contracts.UserNotification, error) {
	_ = ctx
	return s.store.ListUserNotifications(user, limit)
}
func (s *Service) Inbox(ctx context.Context, user string, limit int, before int64) ([]contracts.UserNotification, error) {
	_ = ctx
	return s.store.ListNotificationInbox(user, limit, before)
}
func (s *Service) MarkRead(ctx context.Context, user string, id int64) error {
	_ = ctx
	return s.store.MarkUserNotificationRead(user, id)
}
func (s *Service) MarkAllRead(ctx context.Context, user string) error {
	_ = ctx
	return s.store.MarkAllUserNotificationsRead(user)
}
