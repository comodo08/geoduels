package notifications

import (
	"context"
	"errors"
	"testing"
	"time"

	"geoduels/pkg/contracts"
)

type fakeStore struct {
	err    error
	listed bool
}

func (f *fakeStore) ListUserNotifications(string, int) ([]contracts.UserNotification, error) {
	f.listed = true
	return nil, f.err
}
func (*fakeStore) ListNotificationInbox(string, int, int64) ([]contracts.UserNotification, error) {
	return nil, nil
}
func (*fakeStore) MarkUserNotificationRead(string, int64) error { return nil }
func (*fakeStore) MarkAllUserNotificationsRead(string) error    { return nil }
func (*fakeStore) ClaimPendingNotification(string, time.Time) (contracts.NotificationOutboxItem, bool, error) {
	return contracts.NotificationOutboxItem{}, false, nil
}
func (*fakeStore) MarkNotificationSent(int64) error                      { return nil }
func (*fakeStore) MarkNotificationFailed(int64, time.Time, string) error { return nil }

func TestListDelegatesAndPropagatesError(t *testing.T) {
	want := errors.New("store failed")
	store := &fakeStore{err: want}
	_, err := NewService(store).List(context.Background(), "user", 5)
	if !errors.Is(err, want) || !store.listed {
		t.Fatalf("err=%v listed=%v", err, store.listed)
	}
}
