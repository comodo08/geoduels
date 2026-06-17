package contentfilter

import (
	"errors"
	"strings"
	"unicode"

	goaway "github.com/TwiN/go-away"
)

const (
	MinNicknameLength = 2
	MaxNicknameLength = 14
)

var (
	ErrAbusiveText = errors.New("content failed safety check")
)

// Filter is the swappable content safety boundary for user-authored text.
type Filter interface {
	IsAbusive(text string) bool
}

type goAwayFilter struct{}

func (goAwayFilter) IsAbusive(text string) bool {
	return goaway.IsProfane(text)
}

var defaultFilter Filter = goAwayFilter{}

func SetDefaultFilter(filter Filter) {
	if filter == nil {
		defaultFilter = goAwayFilter{}
		return
	}
	defaultFilter = filter
}

func NormalizeText(raw string, maxRunes int) string {
	body := strings.TrimSpace(raw)
	body = strings.Join(strings.Fields(body), " ")
	if maxRunes <= 0 {
		return body
	}
	runes := []rune(body)
	if len(runes) > maxRunes {
		body = strings.TrimSpace(string(runes[:maxRunes]))
	}
	return body
}

func SanitizeNickname(raw string) (string, error) {
	trimmed := NormalizeText(raw, 0)
	if trimmed == "" {
		return "", errors.New("nickname is required")
	}
	if len([]rune(trimmed)) < MinNicknameLength || len([]rune(trimmed)) > MaxNicknameLength {
		return "", errors.New("nickname must be 2-14 characters")
	}
	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '_' || r == '-' {
			continue
		}
		return "", errors.New("nickname contains invalid characters")
	}
	return trimmed, nil
}

func ValidateNickname(raw string) (string, error) {
	nick, err := SanitizeNickname(raw)
	if err != nil {
		return "", err
	}
	if IsAbusiveText(nick) {
		return "", errors.New("invalid nickname")
	}
	return nick, nil
}

func IsAbusiveText(text string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}
	return defaultFilter.IsAbusive(text)
}

func RejectAbusiveText(values ...string) error {
	for _, value := range values {
		if IsAbusiveText(value) {
			return ErrAbusiveText
		}
	}
	return nil
}
