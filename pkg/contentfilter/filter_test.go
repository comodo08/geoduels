package contentfilter

import "testing"

func TestSanitizeNicknameMaxLength(t *testing.T) {
	if _, err := SanitizeNickname("FourteenChars!"); err == nil {
		t.Fatal("expected invalid character error")
	}
	if nick, err := SanitizeNickname("FourteenChars"); err != nil || nick != "FourteenChars" {
		t.Fatalf("SanitizeNickname 13 chars = %q, %v", nick, err)
	}
	if nick, err := SanitizeNickname("FourteenCharss"); err != nil || nick != "FourteenCharss" {
		t.Fatalf("SanitizeNickname 14 chars = %q, %v", nick, err)
	}
	if _, err := SanitizeNickname("FifteenCharssss"); err == nil {
		t.Fatal("expected nickname over 14 chars to fail")
	}
}

func TestRejectAbusiveTextUsesSwappableFilter(t *testing.T) {
	previous := defaultFilter
	defer SetDefaultFilter(previous)

	SetDefaultFilter(staticFilter{blocked: "blocked phrase"})
	if err := RejectAbusiveText("hello", "blocked phrase"); err != ErrAbusiveText {
		t.Fatalf("RejectAbusiveText error = %v, want %v", err, ErrAbusiveText)
	}
	if err := RejectAbusiveText("hello", "world"); err != nil {
		t.Fatalf("RejectAbusiveText safe text error = %v", err)
	}
}

type staticFilter struct {
	blocked string
}

func (f staticFilter) IsAbusive(text string) bool {
	return text == f.blocked
}
