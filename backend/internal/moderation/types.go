package moderation

import (
	"time"

	"geoduels/pkg/contracts"
)

type ModerationSignalSummary = contracts.ModerationSignalSummary
type ModerationAuditLogEntry = contracts.ModerationAuditLogEntry
type ModerationSubjectProfile = contracts.ModerationSubjectProfile
type ModerationSignalCreated = contracts.ModerationSignalCreated
type ModerationSignalNotificationPayload = contracts.ModerationSignalNotificationPayload

type CreatePlayerReportSignalParams struct {
	MatchID        string
	ReporterUserID string
	ReportedUserID string
	Category       string
	Reason         string
}

type EloRefundSummary struct {
	RefundsIssued int `json:"refundsIssued"`
	TotalRefunded int `json:"totalRefunded"`
}

type CheatingBanSummary struct {
	UserID         string           `json:"userId"`
	Reason         string           `json:"reason,omitempty"`
	Refunds        EloRefundSummary `json:"refunds"`
	IPSignupBanned bool             `json:"ipSignupBanned"`
}

type CommunityPardonSummary struct {
	Eligible int       `json:"eligible"`
	Pardoned int       `json:"pardoned"`
	Cutoff   time.Time `json:"cutoff"`
}

type SignupIPBan struct {
	ID        int64     `json:"id"`
	IPAddress string    `json:"ipAddress"`
	Reason    string    `json:"reason,omitempty"`
	CreatedBy string    `json:"createdBy,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

const (
	moderationProjectionAdvisoryKey = int64(0x67646d6f646572)
	moderationActiveRiskThreshold   = 1.5
)
