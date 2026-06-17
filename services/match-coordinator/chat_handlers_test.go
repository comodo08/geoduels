package main

import (
	"errors"
	"testing"

	"geoduels/pkg/contentfilter"
)

func TestBuildCoordinatorChatMessageRejectsAbusiveText(t *testing.T) {
	contentfilter.SetDefaultFilter(testChatFilter{blocked: "blocked phrase"})
	defer contentfilter.SetDefaultFilter(nil)

	q := &matchCoordinator{}
	_, err := q.buildCoordinatorChatMessage(chatScope{ConversationID: "party:p1", Kind: "party", ID: "p1"}, "u1", "Player", chatClientCommand{
		Type: "chat.send",
		Payload: map[string]any{
			"body": "blocked phrase",
		},
	})
	if !errors.Is(err, contentfilter.ErrAbusiveText) {
		t.Fatalf("buildCoordinatorChatMessage error = %v, want %v", err, contentfilter.ErrAbusiveText)
	}
}

type testChatFilter struct {
	blocked string
}

func (f testChatFilter) IsAbusive(text string) bool {
	return text == f.blocked
}
