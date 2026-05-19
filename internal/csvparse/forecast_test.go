package csvparse

import (
	"math"
	"testing"
	"time"
)

func TestParseForecastTemperature(t *testing.T) {
	f := openTestdata(t, "forecast_tre200h0_small.csv")
	rows, err := ParseForecast(f)
	if err != nil {
		t.Fatalf("ParseForecast: %v", err)
	}
	if got, want := len(rows), 199; got != want {
		t.Fatalf("row count = %d, want %d", got, want)
	}

	// First row in fixture: 1;1;202605162100;-1.3 (UTC).
	r0 := rows[0]
	if r0.PointID != 1 || r0.PointTypeID != 1 {
		t.Errorf("r0 key = (%d,%d), want (1,1)", r0.PointID, r0.PointTypeID)
	}
	wantTS := time.Date(2026, 5, 16, 21, 0, 0, 0, time.UTC)
	if !r0.ValidTime.Equal(wantTS) {
		t.Errorf("r0.ValidTime = %s, want %s", r0.ValidTime, wantTS)
	}
	if r0.ValidTime.Location() != time.UTC {
		t.Errorf("r0.ValidTime.Location = %s, want UTC", r0.ValidTime.Location())
	}
	if math.Abs(r0.Value - -1.3) > 1e-9 {
		t.Errorf("r0.Value = %v, want -1.3", r0.Value)
	}
}

func TestParseForecastPrecipitation(t *testing.T) {
	f := openTestdata(t, "forecast_rre150h0_small.csv")
	rows, err := ParseForecast(f)
	if err != nil {
		t.Fatalf("ParseForecast: %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("no rows parsed")
	}
	// Precipitation values must be non-negative.
	for i, r := range rows {
		if r.Value < 0 {
			t.Errorf("row %d has negative precipitation: %v", i, r.Value)
		}
	}
}

func TestParseForecastIconInteger(t *testing.T) {
	f := openTestdata(t, "forecast_jww003i0_small.csv")
	rows, err := ParseForecast(f)
	if err != nil {
		t.Fatalf("ParseForecast: %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("no rows parsed")
	}
	// Icon codes are small positive integers.
	for i, r := range rows {
		if r.Value <= 0 || r.Value != math.Trunc(r.Value) {
			t.Errorf("row %d unexpected icon value: %v", i, r.Value)
		}
	}
}
