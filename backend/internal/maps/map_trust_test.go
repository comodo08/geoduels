package maps

import "testing"

func TestAutomaticMapCreatorTier(t *testing.T) {
	tests := []struct {
		name       string
		age        int
		favorites  int
		restricted bool
		want       int
	}{
		{name: "new creator stays base", age: 13, favorites: 100, want: mapCreatorTierBase},
		{name: "trusted threshold", age: 14, favorites: 5, want: mapCreatorTierTrusted},
		{name: "established needs more favorites", age: 30, favorites: 49, want: mapCreatorTierTrusted},
		{name: "established threshold", age: 30, favorites: 50, want: mapCreatorTierEstablished},
		{name: "moderation forces base", age: 90, favorites: 500, restricted: true, want: mapCreatorTierBase},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := automaticMapCreatorTier(test.age, test.favorites, test.restricted); got != test.want {
				t.Fatalf("tier = %d, want %d", got, test.want)
			}
		})
	}
}

func TestMapCreatorSpec(t *testing.T) {
	base := specForMapCreatorTier(mapCreatorTierBase)
	if base.maxMaps != 10 || base.maxActiveLocations != 500_000 || base.maxLocationsPerUpload != 300_000 || base.maxUploadedLocationsHour != 300_000 {
		t.Fatalf("unexpected base spec: %+v", base)
	}
	trusted := specForMapCreatorTier(mapCreatorTierTrusted)
	if trusted.favoritesNeeded != 5 || trusted.maxLocationsPerUpload != 600_000 || trusted.accountAgeDays != 14 {
		t.Fatalf("unexpected trusted spec: %+v", trusted)
	}
	established := specForMapCreatorTier(mapCreatorTierEstablished)
	if established.maxMaps != 100 || established.maxActiveLocations != 100_000_000 || established.maxLocationsPerUpload != 1_000_000 || established.maxUploadsPerDay != 30 {
		t.Fatalf("unexpected established spec: %+v", established)
	}
}
