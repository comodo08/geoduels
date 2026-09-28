package accounts

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
)

type memoryStore struct {
	Store
	providers                                                                              map[string]string
	existingUser                                                                           string
	emails                                                                                 []VerifiedEmailUser
	subject                                                                                string
	history                                                                                int
	jobs                                                                                   []string
	banned                                                                                 bool
	enqueueErr                                                                             error
	identityBanned, sessionsRevoked, archived, identitiesDeleted, rolesDeleted, anonymized bool
}

func (m *memoryStore) WithinTx(_ context.Context, fn func(Store) error) error {
	tx := *m
	tx.providers = maps.Clone(m.providers)
	tx.jobs = slices.Clone(m.jobs)
	if err := fn(&tx); err != nil {
		return err
	}
	*m = tx
	return nil
}
func (m *memoryStore) ActiveSeasonID(context.Context) (string, error) { return "season", nil }
func (m *memoryStore) ProviderIdentityBanned(context.Context, string, string) (bool, string, error) {
	return m.banned, "", nil
}
func (m *memoryStore) FindProviderIdentityUser(context.Context, string, string) (string, error) {
	return m.existingUser, nil
}
func (m *memoryStore) FindProviderUserIdentity(_ context.Context, provider, _ string) (string, error) {
	return m.providers[provider], nil
}
func (m *memoryStore) LockOAuthEmail(context.Context, string) error { return nil }
func (m *memoryStore) FindUsersByVerifiedEmail(context.Context, string) ([]VerifiedEmailUser, error) {
	return m.emails, nil
}
func (m *memoryStore) UserIdentityPresence(context.Context, string) (IdentityPresence, error) {
	return IdentityPresence{Found: true, HasIdentity: len(m.providers) > 0}, nil
}
func (m *memoryStore) UpsertRegisteredUser(_ context.Context, id, _, _, _ string) error {
	m.subject = id
	return nil
}
func (m *memoryStore) UpsertIdentityByUserProvider(_ context.Context, id, provider, providerID, _, _, _ string) error {
	m.subject = id
	m.providers[provider] = providerID
	return nil
}
func (m *memoryStore) UpsertIdentityByProviderSubject(ctx context.Context, id, provider, providerID, email, name, avatar string) error {
	return m.UpsertIdentityByUserProvider(ctx, id, provider, providerID, email, name, avatar)
}
func (m *memoryStore) RecordIdentityHistory(context.Context, string, string, string, string, string) error {
	m.history++
	return nil
}
func (m *memoryStore) EnsureAccountRank(context.Context, string, string, string, int) error {
	return nil
}
func (m *memoryStore) EnsureAccountStats(context.Context, string) error { return nil }
func (m *memoryStore) EnsureAccountRankedStats(context.Context, string, string, string) error {
	return nil
}
func (m *memoryStore) GetIdentity(_ context.Context, id string) (Identity, error) {
	providers := make([]string, 0, len(m.providers))
	for p := range m.providers {
		providers = append(providers, p)
	}
	slices.Sort(providers)
	return Identity{Sub: id, IsGuest: len(providers) == 0, LinkedProviders: providers}, nil
}
func (m *memoryStore) CountUserProviders(context.Context, string) (int, error) {
	return len(m.providers), nil
}
func (m *memoryStore) DeleteUserProvider(_ context.Context, _, provider string) (int, error) {
	if _, ok := m.providers[provider]; !ok {
		return 0, nil
	}
	delete(m.providers, provider)
	return 1, nil
}
func (m *memoryStore) MarkIdentityHistoryDeleted(context.Context, string, string) error {
	m.history++
	return nil
}
func (m *memoryStore) ClearUserEmailWithoutGoogle(context.Context, string) error { return nil }
func (m *memoryStore) EnqueueDiscordSync(_ context.Context, action, id string) error {
	if m.enqueueErr != nil {
		return m.enqueueErr
	}
	m.jobs = append(m.jobs, action+":"+id)
	return nil
}
func (m *memoryStore) DeletionUser(context.Context, string) (bool, string, error) {
	return m.banned, "", nil
}
func (m *memoryStore) BanUserOAuthIdentities(context.Context, string, string) error {
	m.identityBanned = true
	return nil
}
func (m *memoryStore) RevokeDeletionSessions(context.Context, string) error {
	m.sessionsRevoked = true
	return nil
}
func (m *memoryStore) ListDeletionDiscordIdentities(context.Context, string) ([]string, error) {
	return []string{"discord-id"}, nil
}
func (m *memoryStore) ArchiveDeletionIdentities(context.Context, string) error {
	m.archived = true
	return nil
}
func (m *memoryStore) DeleteDeletionIdentities(context.Context, string) error {
	m.identitiesDeleted = true
	return nil
}
func (m *memoryStore) DeleteAccountRoles(context.Context, string) error {
	m.rolesDeleted = true
	return nil
}
func (m *memoryStore) AnonymizeDeletedUser(context.Context, string) error {
	m.anonymized = true
	return nil
}

func TestOAuthAccountSelection(t *testing.T) {
	for _, tc := range []struct {
		name, existing, linked, want string
		emails                       []VerifiedEmailUser
		banned, reject               bool
	}{
		{name: "upgrade guest", linked: "guest", want: "guest"},
		{name: "reuse verified email", linked: "guest", want: "registered", emails: []VerifiedEmailUser{{UserID: "registered", HasIdentity: true}}},
		{name: "existing provider wins", existing: "existing", linked: "guest", want: "existing", emails: []VerifiedEmailUser{{UserID: "other", HasIdentity: true}}},
		{name: "ambiguous email", emails: []VerifiedEmailUser{{UserID: "a"}, {UserID: "b"}}, reject: true},
		{name: "blocked new identity", banned: true, reject: true},
		{name: "existing banned account can sign in", existing: "banned", want: "banned", banned: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &memoryStore{providers: map[string]string{}, existingUser: tc.existing, emails: tc.emails, banned: tc.banned}
			identity, err := NewService(store).UpsertProviderIdentity("google", "google-id", "a@example.com", "Name", "", tc.linked)
			if tc.reject {
				if err == nil || store.history != 0 || len(store.providers) != 0 {
					t.Fatalf("rejected login mutated state: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if identity.Sub != tc.want || identity.IsGuest || store.history != 1 {
				t.Fatalf("wrong identity: %+v", identity)
			}
		})
	}
}

func TestUnlinkPreservesLastProviderAndQueuesDiscordCleanup(t *testing.T) {
	store := &memoryStore{providers: map[string]string{"google": "g", "discord": "d"}}
	svc := NewService(store)
	identity, err := svc.UnlinkProviderIdentity("u1", "discord")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(identity.LinkedProviders, []string{"google"}) || !slices.Equal(store.jobs, []string{DiscordSyncActionCleanupRoles + ":d"}) {
		t.Fatalf("identity=%+v jobs=%v", identity, store.jobs)
	}
	if _, err := svc.UnlinkProviderIdentity("u1", "google"); err == nil {
		t.Fatal("removed last sign-in method")
	}
	if store.providers["google"] != "g" {
		t.Fatal("last provider was deleted")
	}
}

func TestUnlinkFailureKeepsProvider(t *testing.T) {
	failure := errors.New("queue unavailable")
	store := &memoryStore{providers: map[string]string{"google": "g", "discord": "d"}, enqueueErr: failure}
	if _, err := NewService(store).UnlinkProviderIdentity("u1", "discord"); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if len(store.providers) != 2 || store.history != 0 || len(store.jobs) != 0 {
		t.Fatal("failed unlink leaked changes")
	}
}

func TestDeletionPreservesBanAndRevokesAccess(t *testing.T) {
	store := &memoryStore{banned: true}
	if err := NewService(store).DeleteAccount("u1"); err != nil {
		t.Fatal(err)
	}
	if !store.identityBanned || !store.sessionsRevoked || !store.archived || !store.identitiesDeleted || !store.rolesDeleted || !store.anonymized {
		t.Fatal("incomplete account deletion")
	}
	if !slices.Equal(store.jobs, []string{DiscordSyncActionCleanupRoles + ":discord-id"}) {
		t.Fatal("missing Discord cleanup")
	}
}
