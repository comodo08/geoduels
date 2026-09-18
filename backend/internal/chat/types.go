package chat

import (
	"time"

	"geoduels/pkg/contracts"
)

type ChatMessage = contracts.ChatMessage

type ChatRestriction struct {
	ActionType string
	ReasonCode string
	ReasonNote string
	EndsAt     time.Time
}
