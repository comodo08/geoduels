package main

import (
	"geoduels/internal/accounts"
	"geoduels/internal/admin"
	"geoduels/internal/authsession"
	"geoduels/internal/badges"
	"geoduels/internal/chat"
	"geoduels/internal/content"
	"geoduels/internal/maps"
	"geoduels/internal/matches"
	"geoduels/internal/moderation"
	"geoduels/internal/parties"
	"geoduels/internal/profiles"
	"geoduels/internal/seasons"
	socialdomain "geoduels/internal/social"
)

type testRepositories interface {
	accounts.Store
	authsession.Store
	profiles.Store
	badges.Store
	matches.Store
	moderation.Store
	admin.Store
	content.Store
	seasons.Store
	maps.Store
	chat.Store
	parties.Store
	socialdomain.Store
}

func withTestRepositories(store testRepositories) func(*api) {
	return func(a *api) {
		a.accounts = store
		a.sessions = store
		a.profiles = store
		a.badges = store
		a.matchStore = store
		a.moderation = store
		a.admin = store
		a.content = store
		a.seasons = store
		a.gameplayMaps = store
		a.runtimeStore = store
		a.chatStore = store
		a.parties = store
		a.social = socialdomain.NewService(store)
	}
}
