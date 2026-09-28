package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/riverqueue/river"

	"geoduels/internal/content"
	"geoduels/internal/jobs"
	"geoduels/internal/storekit"
	"geoduels/pkg/observability"
	db "geoduels/pkg/persistence/sqlc/db"
)

type discordWebhookMessage struct {
	Username string         `json:"username,omitempty"`
	Embeds   []discordEmbed `json:"embeds"`
}

type discordEmbed struct {
	Title     string              `json:"title"`
	Color     int                 `json:"color"`
	Fields    []discordEmbedField `json:"fields"`
	Timestamp string              `json:"timestamp,omitempty"`
}

type discordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// moderationNotifyWorker delivers a queued moderation-signal webhook.
type moderationNotifyWorker struct {
	river.WorkerDefaults[jobs.ModerationNotifyArgs]
	queries    *db.Queries
	content    content.Store
	httpClient *http.Client
}

func (w *moderationNotifyWorker) Work(ctx context.Context, job *river.Job[jobs.ModerationNotifyArgs]) error {
	if job.Args.SignalID <= 0 || w.queries == nil {
		return nil
	}
	row, err := w.queries.GetSignalNotificationPayload(ctx, job.Args.SignalID)
	if err != nil {
		return err
	}
	settings, err := w.content.GetModerationSettings()
	if err != nil {
		return err
	}
	if strings.TrimSpace(settings.DiscordWebhookURL) == "" {
		return nil
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, err := sendDiscordReportNotification(timeoutCtx, w.httpClient, settings.DiscordWebhookURL, row); err != nil {
		return err
	}
	observability.Log("info", "moderation signal notification sent", map[string]any{"signal_id": row.SignalID, "subject_user_id": storekit.UUIDVal(row.SubjectUserID)})
	return nil
}

func sendDiscordReportNotification(ctx context.Context, client *http.Client, webhookURL string, payload db.GetSignalNotificationPayloadRow) (time.Duration, error) {
	if client == nil {
		client = http.DefaultClient
	}
	body, err := json.Marshal(discordWebhookMessage{
		Username: "GeoDuels Moderation",
		Embeds: []discordEmbed{{
			Title: "Moderation signal needs review",
			Color: 0xff4d4f,
			Fields: []discordEmbedField{
				{Name: "Signal", Value: fmt.Sprintf("#%d", payload.SignalID), Inline: true},
				{Name: "Severity", Value: strings.ToUpper(string(payload.Severity)), Inline: true},
				{Name: "Subject", Value: discordUserValue(storekit.TextVal(payload.SubjectDisplayName), storekit.UUIDVal(payload.SubjectUserID)), Inline: false},
				{Name: "Evidence", Value: fmt.Sprintf("%s / %s", strings.ToUpper(payload.ReasonCode), strings.ToUpper(string(payload.EvidenceStrength))), Inline: false},
			},
			Timestamp: payload.OccurredAt.Time.UTC().Format(time.RFC3339),
		}},
	})
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return 0, nil
	}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	retryAfter := discordRetryAfter(resp, raw)
	msg := strings.TrimSpace(string(raw))
	if msg == "" {
		msg = resp.Status
	}
	return retryAfter, fmt.Errorf("discord webhook returned %s: %s", resp.Status, msg)
}

func discordUserValue(name, userID string) string {
	name = strings.TrimSpace(name)
	userID = strings.TrimSpace(userID)
	if name == "" {
		name = userID
	}
	if userID == "" || userID == name {
		return discordFieldValue(name, "Unknown")
	}
	return discordFieldValue(name+"\n`"+userID+"`", "Unknown")
}

func discordFieldValue(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	if len(value) > 1000 {
		value = value[:997] + "..."
	}
	return value
}

func discordRetryAfter(resp *http.Response, rawBody []byte) time.Duration {
	if resp == nil {
		return 0
	}
	if header := strings.TrimSpace(resp.Header.Get("Retry-After")); header != "" {
		if seconds, err := strconv.ParseFloat(header, 64); err == nil && seconds > 0 {
			return time.Duration(seconds * float64(time.Second))
		}
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		return 0
	}
	var body struct {
		RetryAfter float64 `json:"retry_after"`
	}
	if err := json.Unmarshal(rawBody, &body); err != nil {
		return 0
	}
	if body.RetryAfter <= 0 {
		return 0
	}
	return time.Duration(body.RetryAfter * float64(time.Second))
}
