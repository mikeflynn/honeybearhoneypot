package geo

import "testing"

func TestLookupKnownIP(t *testing.T) {
	if err := Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Close()

	// 8.8.8.8 (Google DNS) is a stable, well-known IP that every GeoIP DB
	// resolves to the United States.
	g, err := Lookup("8.8.8.8")
	if err != nil {
		t.Fatalf("Lookup error: %v", err)
	}
	if g == nil {
		t.Fatal("expected non-nil Geo for 8.8.8.8")
	}
	if g.CountryCode != "US" {
		t.Errorf("expected CountryCode=US, got %q", g.CountryCode)
	}
	if g.Lat == 0 && g.Lon == 0 {
		t.Errorf("expected non-zero lat/lon (US centroid), got %+v", g)
	}
}

func TestLookupPrivateIP(t *testing.T) {
	if err := Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Close()

	g, err := Lookup("10.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g != nil {
		t.Errorf("expected nil for private IP, got %+v", g)
	}
}

func TestLookupInvalidIP(t *testing.T) {
	if err := Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Close()

	g, err := Lookup("not-an-ip")
	if err == nil {
		t.Errorf("expected error for invalid IP, got %+v", g)
	}
}

func TestCentroidsCoverage(t *testing.T) {
	for _, code := range []string{"US", "CN", "RU", "DE", "BR", "IN", "JP", "GB", "AU", "ZA"} {
		if _, ok := Centroids[code]; !ok {
			t.Errorf("Centroids missing %q", code)
		}
	}
	if len(Centroids) < 200 {
		t.Errorf("Centroids has only %d entries; expected ~245", len(Centroids))
	}
}
