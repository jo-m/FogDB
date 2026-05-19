package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"jo-m.ch/go/nebeltracker/internal/csvparse"
)

// openTestDB creates a temporary SQLite database, applies all migrations, and
// returns the open *sql.DB. The database file is cleaned up automatically when
// the test ends via t.TempDir.
//
// Parameters:
//   - t: the testing instance whose cleanup mechanism owns the temp dir.
//
// Returns the open database handle; the test is fatally failed on any error.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.sqlite")
	db, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("openTestDB: Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// queryPragmaInt executes "PRAGMA <name>" and scans the single integer result.
func queryPragmaInt(t *testing.T, db *sql.DB, name string) int64 {
	t.Helper()
	var v int64
	// #nosec G202 -- pragma name is a test-internal constant, not user input.
	if err := db.QueryRowContext(context.Background(), "PRAGMA "+name).Scan(&v); err != nil {
		t.Fatalf("PRAGMA %s: %v", name, err)
	}
	return v
}

// queryPragmaText executes "PRAGMA <name>" and scans the single text result.
func queryPragmaText(t *testing.T, db *sql.DB, name string) string {
	t.Helper()
	var v string
	// #nosec G202 -- pragma name is a test-internal constant, not user input.
	if err := db.QueryRowContext(context.Background(), "PRAGMA "+name).Scan(&v); err != nil {
		t.Fatalf("PRAGMA %s: %v", name, err)
	}
	return v
}

func TestPragmas(t *testing.T) {
	db := openTestDB(t)

	// journal_mode returns a text value; SQLite normalises it to lowercase.
	if got := queryPragmaText(t, db, "journal_mode"); got != "wal" {
		t.Errorf("journal_mode: got %q, want %q", got, "wal")
	}
	// synchronous: OFF=0, NORMAL=1, FULL=2, EXTRA=3.
	if got := queryPragmaInt(t, db, "synchronous"); got != 1 {
		t.Errorf("synchronous: got %d, want 1 (NORMAL)", got)
	}
	// temp_store: DEFAULT=0, FILE=1, MEMORY=2.
	if got := queryPragmaInt(t, db, "temp_store"); got != 2 {
		t.Errorf("temp_store: got %d, want 2 (MEMORY)", got)
	}
	// foreign_keys: OFF=0, ON=1.
	if got := queryPragmaInt(t, db, "foreign_keys"); got != 1 {
		t.Errorf("foreign_keys: got %d, want 1 (ON)", got)
	}
}

func TestUpsertParameters(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	params := []csvparse.Parameter{
		{
			Shortname: "tre200h0", DescriptionDE: "Temperatur 2 m", DescriptionFR: "Temp 2 m",
			DescriptionIT: "Temp 2 m", DescriptionEN: "Temperature 2 m",
			GroupDE: "Temperatur", GroupFR: "Temp", GroupIT: "Temp", GroupEN: "Temp",
			Granularity: "h", Decimals: 1, Datatype: "float", Unit: "degC",
		},
		{
			Shortname: "rre150h0", DescriptionDE: "Niederschlag", DescriptionFR: "Precip",
			DescriptionIT: "Precip", DescriptionEN: "Precipitation",
			GroupDE: "Niederschlag", GroupFR: "Precip", GroupIT: "Precip", GroupEN: "Precip",
			Granularity: "h", Decimals: 1, Datatype: "float", Unit: "mm",
		},
	}

	ids, err := UpsertParameters(ctx, db, params)
	if err != nil {
		t.Fatalf("UpsertParameters: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 parameter IDs, got %d", len(ids))
	}
	for _, name := range []string{"tre200h0", "rre150h0"} {
		if ids[name] == 0 {
			t.Errorf("parameter %q has id 0", name)
		}
	}

	// A second upsert with updated description must not duplicate rows.
	params[0].DescriptionEN = "Air temperature 2 m"
	ids2, err := UpsertParameters(ctx, db, params)
	if err != nil {
		t.Fatalf("UpsertParameters (idempotent): %v", err)
	}
	if len(ids2) != 2 {
		t.Fatalf("expected 2 parameter IDs after re-upsert, got %d", len(ids2))
	}
}

func TestUpsertForecastsRoundtrip(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	params := []csvparse.Parameter{{
		Shortname: "tre200h0", DescriptionDE: "Temp", DescriptionFR: "Temp",
		DescriptionIT: "Temp", DescriptionEN: "Temp",
		GroupDE: "Temp", GroupFR: "Temp", GroupIT: "Temp", GroupEN: "Temp",
		Granularity: "h", Decimals: 1, Datatype: "float", Unit: "degC",
	}}
	paramIDs, err := UpsertParameters(ctx, db, params)
	if err != nil {
		t.Fatalf("UpsertParameters: %v", err)
	}

	locs := []csvparse.Location{{
		PointID: 1, PointTypeID: 1, PointName: "Zuerich", StationAbbr: "ZUE",
		PointTypeDE: "Stadt", PointTypeFR: "Ville", PointTypeIT: "Citta", PointTypeEN: "City",
		HeightMASL: 408, WGS84Lat: 47.371935, WGS84Lon: 8.539336,
	}}
	locIDs, err := UpsertLocations(ctx, db, locs)
	if err != nil {
		t.Fatalf("UpsertLocations: %v", err)
	}

	locKey := LocationKey{PointID: 1, PointTypeID: 1}
	ts := time.Date(2026, 5, 19, 4, 0, 0, 0, time.UTC)
	forecasts := []ForecastRow{{
		LocationID:  locIDs[locKey],
		ParameterID: paramIDs["tre200h0"],
		Timestamp:   ts,
		Value:       12.5,
	}}
	if err := UpsertForecasts(ctx, db, forecasts); err != nil {
		t.Fatalf("UpsertForecasts: %v", err)
	}

	// Read back the stored value.
	var gotValue float64
	var gotTS string
	err = db.QueryRowContext(ctx,
		`SELECT value, timestamp FROM forecasts WHERE location_id = ? AND parameter_id = ?`,
		locIDs[locKey], paramIDs["tre200h0"],
	).Scan(&gotValue, &gotTS)
	if err != nil {
		t.Fatalf("SELECT forecast: %v", err)
	}
	if gotValue != 12.5 {
		t.Errorf("value: got %v, want 12.5", gotValue)
	}
	wantTS := ts.UTC().Format(time.RFC3339)
	if gotTS != wantTS {
		t.Errorf("timestamp: got %q, want %q", gotTS, wantTS)
	}

	// Upsert the same key with a new value; row count must stay at 1.
	forecasts[0].Value = 15.0
	if err := UpsertForecasts(ctx, db, forecasts); err != nil {
		t.Fatalf("UpsertForecasts (update): %v", err)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM forecasts`).Scan(&count); err != nil {
		t.Fatalf("COUNT forecasts: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 forecast row after upsert, got %d", count)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT value FROM forecasts WHERE location_id = ? AND parameter_id = ?`,
		locIDs[locKey], paramIDs["tre200h0"],
	).Scan(&gotValue); err != nil {
		t.Fatalf("SELECT updated forecast: %v", err)
	}
	if gotValue != 15.0 {
		t.Errorf("updated value: got %v, want 15.0", gotValue)
	}
}

func TestForeignKeyEnforcement(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	// Inserting a forecast that references non-existent FKs must fail because
	// foreign_keys = ON is set on every connection.
	err := UpsertForecasts(ctx, db, []ForecastRow{{
		LocationID:  9999,
		ParameterID: 9999,
		Timestamp:   time.Now().UTC(),
		Value:       0,
	}})
	if err == nil {
		t.Fatal("expected foreign-key violation, got nil error")
	}
}
