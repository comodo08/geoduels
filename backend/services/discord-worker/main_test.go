package main

import (
	"testing"

	"geoduels/internal/accounts"
	"geoduels/internal/content"
)

func TestDiscordSyncActionForLinkStatePreventsStaleCleanup(t *testing.T) {
	if got := discordSyncActionForLinkState(accounts.DiscordSyncActionCleanupRoles, true); got != accounts.DiscordSyncActionSync {
		t.Fatalf("linked cleanup action = %q, want sync", got)
	}
	if got := discordSyncActionForLinkState(accounts.DiscordSyncActionCleanupRoles, false); got != accounts.DiscordSyncActionCleanupRoles {
		t.Fatalf("unlinked cleanup action = %q, want cleanup", got)
	}
}

func TestRankRoleForMMR(t *testing.T) {
	settings := content.DiscordIntegrationSettings{
		Elo1000RoleID: "role-1000",
		Elo1500RoleID: "role-1500",
		Elo2000RoleID: "role-2000",
	}
	tests := []struct {
		mmr  int
		want string
	}{
		{999, ""},
		{1000, "role-1000"},
		{1499, "role-1000"},
		{1500, "role-1500"},
		{1999, "role-1500"},
		{2000, "role-2000"},
	}
	for _, test := range tests {
		if got := rankRoleForMMR(settings, test.mmr); got != test.want {
			t.Errorf("rankRoleForMMR(%d) = %q, want %q", test.mmr, got, test.want)
		}
	}
}
