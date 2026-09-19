package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"geoduels/pkg/contracts"
	"github.com/labstack/echo/v4"
)

// The v2 party socket carries commands as well as authoritative state events.
// Commands are processed serially per connection and are never replayed on reconnect.
type partyCommand struct {
	RequestID string          `json:"requestId"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type partyCommandResult struct {
	RequestID string `json:"requestId"`
	OK        bool   `json:"ok"`
	Error     string `json:"error,omitempty"`
}

type partySettingsRequest struct {
	Mode   contracts.MatchMode   `json:"mode"`
	Config contracts.MatchConfig `json:"config"`
}

func partyErrorMessage(err error) string {
	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) {
		return fmt.Sprint(httpErr.Message)
	}
	return err.Error()
}

func (q *matchCoordinator) executePartyCommand(ctx context.Context, partyID, userID string, command partyCommand) error {
	// Membership, ownership and account eligibility can change after the handshake.
	identity, err := q.accounts.GetIdentity(userID)
	if err != nil {
		return errors.New("identity not found")
	}
	if identity.IsBanned {
		return errors.New("user is banned")
	}
	if identity.NicknameRequired {
		return errors.New("nickname required")
	}
	if identity.AuthMigrationRequired {
		return errors.New("connect discord to continue")
	}
	snap, found, err := q.parties.GetPartyByID(partyID)
	if err != nil {
		return errors.New("party unavailable")
	}
	if !found || !partyHasMember(snap, userID) {
		return errors.New("party membership required")
	}
	switch command.Type {
	case "team":
		var req contracts.PartyTeamRequest
		if json.Unmarshal(command.Payload, &req) != nil {
			return errors.New("invalid payload")
		}
		_, err = q.parties.SetPartyMemberTeam(partyID, userID, strings.TrimSpace(req.TeamID))
	case "settings":
		var req partySettingsRequest
		if json.Unmarshal(command.Payload, &req) != nil {
			return errors.New("invalid payload")
		}
		_, err = q.setPartySettings(ctx, partyID, userID, req)
		return err
	case "kick", "transfer_owner":
		var req contracts.PartyMemberRequest
		if json.Unmarshal(command.Payload, &req) != nil {
			return errors.New("invalid payload")
		}
		if command.Type == "kick" {
			_, err = q.parties.KickPartyMember(partyID, userID, strings.TrimSpace(req.UserID))
		} else {
			_, err = q.parties.TransferPartyOwner(partyID, userID, strings.TrimSpace(req.UserID))
		}
	case "leave":
		_, err = q.parties.LeaveParty(partyID, userID)
	case "start":
		_, err = q.startPartyMatch(ctx, partyID, userID)
		return err
	default:
		return errors.New("unknown party command")
	}
	if err == nil {
		q.publishPartyChanged(ctx, partyID)
	}
	return err
}
