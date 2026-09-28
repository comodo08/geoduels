package staff

import (
	"context"
	"time"
)

// CheatingBanSummary reports the outcome of a cheating ban.
type CheatingBanSummary struct {
	UserID         string           `json:"userId"`
	Reason         string           `json:"reason,omitempty"`
	Refunds        EloRefundSummary `json:"refunds"`
	IPSignupBanned bool             `json:"ipSignupBanned"`
}

// EloRefundSummary counts rating refunds issued during enforcement.
type EloRefundSummary struct {
	RefundsIssued int `json:"refundsIssued"`
	TotalRefunded int `json:"totalRefunded"`
}

// CommunityPardonSummary reports a pardon preview or execution.
type CommunityPardonSummary struct {
	Eligible int       `json:"eligible"`
	Pardoned int       `json:"pardoned"`
	Cutoff   time.Time `json:"cutoff"`
}

// SignupIPBan is an active IP signup block.
type SignupIPBan struct {
	ID        int64     `json:"id"`
	IPAddress string    `json:"ipAddress"`
	Reason    string    `json:"reason,omitempty"`
	CreatedBy string    `json:"createdBy,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// Cadence is the fixed weekly curation window.
const CurationWindow = 7 * 24 * time.Hour

// Nomination is one Map-of-the-Week submission.
type Nomination struct {
	ID           int64     `json:"id"`
	MapID        string    `json:"mapId"`
	Name         string    `json:"name"`
	ThumbnailKey string    `json:"thumbnailKey"`
	AuthorName   string    `json:"authorName"`
	Likes        int       `json:"likes"`
	Liked        bool      `json:"liked"`
	NominatedAt  time.Time `json:"nominatedAt"`
}

// CurationPage is a page of nominations plus the closing time.
type CurationPage struct {
	Items    []Nomination `json:"items"`
	Total    int          `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
	ClosesAt time.Time    `json:"closesAt"`
}

// RiskSignal is one detector finding.
type RiskSignal struct {
	SubjectUserID    string         `json:"subjectUserId"`
	SignalType       string         `json:"signalType"`
	DetectorKey      string         `json:"detectorKey"`
	DetectorVersion  string         `json:"detectorVersion"`
	Severity         string         `json:"severity"`
	EvidenceStrength string         `json:"evidenceStrength"`
	ReasonCode       string         `json:"reasonCode"`
	RecommendedQueue bool           `json:"recommendedQueue"`
	Score            float64        `json:"score"`
	Payload          map[string]any `json:"payload,omitempty"`
	OccurredAt       time.Time      `json:"occurredAt"`
}

// RiskGuessEvent is one guess observation sent to the detector.
type RiskGuessEvent struct {
	MatchID     string    `json:"matchId"`
	RoundNumber int       `json:"roundNumber"`
	Score       int       `json:"score"`
	GuessMS     int       `json:"guessMs"`
	Evidence    float64   `json:"evidence"`
	OccurredAt  time.Time `json:"occurredAt"`
}

// RiskPlayer is per-player history sent to the detector.
type RiskPlayer struct {
	UserID        string           `json:"userId"`
	CurrentRating int              `json:"currentRating,omitempty"`
	RankedGames   int              `json:"rankedGames,omitempty"`
	Events        []RiskGuessEvent `json:"events"`
}

// RiskRequest is the detector analyze request.
type RiskRequest struct {
	RequestID    string       `json:"requestId,omitempty"`
	MatchID      string       `json:"matchId"`
	FactsVersion string       `json:"factsVersion"`
	GeneratedAt  time.Time    `json:"generatedAt"`
	Players      []RiskPlayer `json:"players"`
}

// RiskResponse is the detector analyze response.
type RiskResponse struct {
	DetectorVersion string       `json:"detectorVersion"`
	FactsVersion    string       `json:"factsVersion"`
	Signals         []RiskSignal `json:"signals"`
}

// RiskEngine analyzes match facts for integrity signals.
type RiskEngine interface {
	Analyze(ctx context.Context, req RiskRequest) (RiskResponse, error)
	Enabled() bool
}

// ReportEligibility contains the stored facts used to authorize a player report.
type ReportEligibility struct{ TargetParticipated, ReporterMuted bool }
type PlayerReport struct {
	MatchID, ReporterID, SubjectID, Category, Reason, Severity string
	Score                                                      float64
}
