package csvparse

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func openTestdata(t *testing.T, name string) *os.File {
	t.Helper()
	path := filepath.Join("testdata", name)
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func TestParseParameters(t *testing.T) {
	f := openTestdata(t, "meta_parameters.csv")
	params, err := ParseParameters(f)
	if err != nil {
		t.Fatalf("ParseParameters: %v", err)
	}
	if got, want := len(params), 32; got != want {
		t.Fatalf("row count = %d, want %d", got, want)
	}

	// First row sanity check.
	p0 := params[0]
	if p0.Shortname != "dkl010h0" {
		t.Errorf("p0.Shortname = %q, want %q", p0.Shortname, "dkl010h0")
	}
	if !strings.Contains(p0.DescriptionDE, "Windrichtung") {
		t.Errorf("p0.DescriptionDE missing 'Windrichtung': %q", p0.DescriptionDE)
	}
	if p0.GroupEN != "Wind" {
		t.Errorf("p0.GroupEN = %q, want Wind", p0.GroupEN)
	}
	if p0.Datatype != "Integer" {
		t.Errorf("p0.Datatype = %q, want Integer", p0.Datatype)
	}
	if p0.Decimals != 0 {
		t.Errorf("p0.Decimals = %d, want 0", p0.Decimals)
	}
	// Unit for wind direction is the degrees sign in Latin1 (0xB0); after
	// decoding it must be the UTF-8 degree sign.
	if p0.Unit != "°" {
		t.Errorf("p0.Unit = %q, want degree sign", p0.Unit)
	}

	// Verify Latin1 decoding produced proper umlauts somewhere in the file.
	var sawUmlaut bool
	for _, p := range params {
		if strings.ContainsAny(p.DescriptionDE, "äöüÄÖÜß") {
			sawUmlaut = true
			break
		}
	}
	if !sawUmlaut {
		t.Errorf("expected at least one German umlaut in descriptions; Latin1 decoding likely failed")
	}

	// Find a known parameter we care about.
	var foundTre bool
	for _, p := range params {
		if p.Shortname == "tre200h0" {
			foundTre = true
			if p.Datatype != "Float" {
				t.Errorf("tre200h0.Datatype = %q, want Float", p.Datatype)
			}
			if !strings.Contains(p.Unit, "C") { // °C
				t.Errorf("tre200h0.Unit = %q, want to contain C", p.Unit)
			}
		}
	}
	if !foundTre {
		t.Errorf("did not find tre200h0 in parameters")
	}
}

func TestParseLocations(t *testing.T) {
	f := openTestdata(t, "meta_point.csv")
	locs, err := ParseLocations(f)
	if err != nil {
		t.Fatalf("ParseLocations: %v", err)
	}
	if got, want := len(locs), 5629; got != want {
		t.Fatalf("row count = %d, want %d", got, want)
	}

	// First row (Arosa station).
	l0 := locs[0]
	if l0.PointID != 1 || l0.PointTypeID != 1 {
		t.Errorf("l0 key = (%d,%d), want (1,1)", l0.PointID, l0.PointTypeID)
	}
	if l0.StationAbbr != "ARO" {
		t.Errorf("l0.StationAbbr = %q, want ARO", l0.StationAbbr)
	}
	if l0.PointName != "Arosa" {
		t.Errorf("l0.PointName = %q, want Arosa", l0.PointName)
	}
	if l0.PointTypeEN != "Station" {
		t.Errorf("l0.PointTypeEN = %q, want Station", l0.PointTypeEN)
	}
	if math.Abs(l0.HeightMASL-1878.0) > 1e-6 {
		t.Errorf("l0.HeightMASL = %v, want 1878.0", l0.HeightMASL)
	}
	if math.Abs(l0.WGS84Lat-46.792661) > 1e-6 {
		t.Errorf("l0.WGS84Lat = %v, want ~46.792661", l0.WGS84Lat)
	}
	if math.Abs(l0.WGS84Lon-9.679014) > 1e-6 {
		t.Errorf("l0.WGS84Lon = %v, want ~9.679014", l0.WGS84Lon)
	}

	// Ensure we see all three point types.
	types := map[int64]bool{}
	for _, l := range locs {
		types[l.PointTypeID] = true
	}
	for _, want := range []int64{1, 2, 3} {
		if !types[want] {
			t.Errorf("expected to see point_type_id %d", want)
		}
	}
}
