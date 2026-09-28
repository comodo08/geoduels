package staff

import "strings"

func normalizeReportCategory(category string) string {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "cheating", "profile", "harassment", "boosting":
		return strings.ToLower(strings.TrimSpace(category))
	default:
		return "other"
	}
}

func reportSeverity(category string) string {
	switch category {
	case "cheating", "boosting", "harassment":
		return "medium"
	default:
		return "low"
	}
}

func reportScore(category string) float64 {
	switch category {
	case "cheating", "boosting":
		return 2
	case "harassment":
		return 1.5
	default:
		return 1
	}
}
