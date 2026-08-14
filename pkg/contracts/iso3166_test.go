package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestIsCountryCodeAllowed(t *testing.T) {
	cases := []struct {
		code string
		want bool
	}{
		{"FR", true},
		{"US", true},
		{"ZW", true},
		{"", false},
		{"fr", false},
		{"F", false},
		{"FRA", false},
		{"XX", false},
		{"EU", false},
		{"UN", false},
		{"AA", false},
		{"ZZ", false},
	}
	for _, tc := range cases {
		if got := IsCountryCodeAllowed(tc.code); got != tc.want {
			t.Errorf("IsCountryCodeAllowed(%q) = %v, want %v", tc.code, got, tc.want)
		}
	}
}

func TestGeneratedAllowlistMatchesSourceJSON(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "datasets-config", "iso3166-alpha2.json"))
	if err != nil {
		t.Fatalf("reading source JSON: %v", err)
	}
	var source struct {
		Countries []struct {
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"countries"`
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		t.Fatalf("parsing source JSON: %v", err)
	}
	if len(source.Countries) == 0 {
		t.Fatal("source JSON contains no countries")
	}
	expected := make(map[string]struct{}, len(source.Countries))
	for _, entry := range source.Countries {
		expected[entry.Code] = struct{}{}
		if entry.Name == "" {
			t.Errorf("source entry %q has no name", entry.Code)
		}
		if !IsCountryCodeAllowed(entry.Code) {
			t.Errorf("IsCountryCodeAllowed(%q) = false, want true for source code", entry.Code)
		}
	}
	if len(allowedCountryCodes) != len(expected) {
		t.Errorf("allowlist has %d codes, source JSON has %d", len(allowedCountryCodes), len(expected))
	}
	for code := range expected {
		if _, ok := allowedCountryCodes[code]; !ok {
			t.Errorf("allowlist is missing source code %q", code)
		}
	}
	for code := range allowedCountryCodes {
		if _, ok := expected[code]; !ok {
			t.Errorf("allowlist contains %q, which is not in the source JSON", code)
		}
	}
}
