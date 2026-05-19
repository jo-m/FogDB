package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"jo-m.ch/go/fogdb/internal/csvparse"
)

// UpsertParameters writes/updates one row per entry into the parameters table
// using parameter_shortname as the upsert key. Returns a map from shortname to
// the internal numeric primary key.
func UpsertParameters(ctx context.Context, sqlDB *sql.DB, params []csvparse.Parameter) (map[string]int64, error) {
	slog.Info("upserting parameters", "count", len(params))
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const upsertSQL = `
		INSERT INTO parameters (
			parameter_shortname,
			parameter_description_de, parameter_description_fr,
			parameter_description_it, parameter_description_en,
			parameter_group_de, parameter_group_fr,
			parameter_group_it, parameter_group_en,
			parameter_granularity, parameter_decimals,
			parameter_datatype, parameter_unit
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (parameter_shortname) DO UPDATE SET
			parameter_description_de = excluded.parameter_description_de,
			parameter_description_fr = excluded.parameter_description_fr,
			parameter_description_it = excluded.parameter_description_it,
			parameter_description_en = excluded.parameter_description_en,
			parameter_group_de = excluded.parameter_group_de,
			parameter_group_fr = excluded.parameter_group_fr,
			parameter_group_it = excluded.parameter_group_it,
			parameter_group_en = excluded.parameter_group_en,
			parameter_granularity = excluded.parameter_granularity,
			parameter_decimals = excluded.parameter_decimals,
			parameter_datatype = excluded.parameter_datatype,
			parameter_unit = excluded.parameter_unit
	`

	stmt, err := tx.PrepareContext(ctx, upsertSQL)
	if err != nil {
		return nil, fmt.Errorf("prepare: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, p := range params {
		if _, err := stmt.ExecContext(ctx,
			p.Shortname,
			p.DescriptionDE, p.DescriptionFR, p.DescriptionIT, p.DescriptionEN,
			p.GroupDE, p.GroupFR, p.GroupIT, p.GroupEN,
			p.Granularity, p.Decimals,
			p.Datatype, p.Unit,
		); err != nil {
			return nil, fmt.Errorf("upsert parameter %q: %w", p.Shortname, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return loadParameterIDs(ctx, sqlDB)
}

// loadParameterIDs returns a map from parameter_shortname to the internal id.
func loadParameterIDs(ctx context.Context, sqlDB *sql.DB) (map[string]int64, error) {
	rows, err := sqlDB.QueryContext(ctx, `SELECT id, parameter_shortname FROM parameters`)
	if err != nil {
		return nil, fmt.Errorf("query parameter ids: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make(map[string]int64)
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("scan parameter id: %w", err)
		}
		out[name] = id
	}
	return out, rows.Err()
}

// LocationKey identifies a point uniquely (point_id, point_type_id).
type LocationKey struct {
	PointID     int64
	PointTypeID int64
}

// UpsertLocations writes/updates one row per location into the locations
// table. Returns a map from (point_id, point_type_id) to the internal id.
func UpsertLocations(ctx context.Context, sqlDB *sql.DB, locs []csvparse.Location) (map[LocationKey]int64, error) {
	slog.Info("upserting locations", "count", len(locs))
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const upsertSQL = `
		INSERT INTO locations (
			point_id, point_type_id, station_abbr, postal_code, point_name,
			point_type_de, point_type_fr, point_type_it, point_type_en,
			point_height_masl,
			point_coordinates_lv95_east, point_coordinates_lv95_north,
			point_coordinates_wgs84_lat, point_coordinates_wgs84_lon
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (point_id, point_type_id) DO UPDATE SET
			station_abbr = excluded.station_abbr,
			postal_code = excluded.postal_code,
			point_name = excluded.point_name,
			point_type_de = excluded.point_type_de,
			point_type_fr = excluded.point_type_fr,
			point_type_it = excluded.point_type_it,
			point_type_en = excluded.point_type_en,
			point_height_masl = excluded.point_height_masl,
			point_coordinates_lv95_east = excluded.point_coordinates_lv95_east,
			point_coordinates_lv95_north = excluded.point_coordinates_lv95_north,
			point_coordinates_wgs84_lat = excluded.point_coordinates_wgs84_lat,
			point_coordinates_wgs84_lon = excluded.point_coordinates_wgs84_lon
	`

	stmt, err := tx.PrepareContext(ctx, upsertSQL)
	if err != nil {
		return nil, fmt.Errorf("prepare: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, l := range locs {
		if _, err := stmt.ExecContext(ctx,
			l.PointID, l.PointTypeID, l.StationAbbr, l.PostalCode, l.PointName,
			l.PointTypeDE, l.PointTypeFR, l.PointTypeIT, l.PointTypeEN,
			nullableFloat(l.HeightMASL),
			nullableFloat(l.LV95East), nullableFloat(l.LV95North),
			nullableFloat(l.WGS84Lat), nullableFloat(l.WGS84Lon),
		); err != nil {
			return nil, fmt.Errorf("upsert location (%d,%d): %w", l.PointID, l.PointTypeID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return loadLocationIDs(ctx, sqlDB)
}

// loadLocationIDs returns a map from (point_id, point_type_id) to id.
func loadLocationIDs(ctx context.Context, sqlDB *sql.DB) (map[LocationKey]int64, error) {
	rows, err := sqlDB.QueryContext(ctx, `SELECT id, point_id, point_type_id FROM locations`)
	if err != nil {
		return nil, fmt.Errorf("query location ids: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make(map[LocationKey]int64)
	for rows.Next() {
		var id, pid, ptid int64
		if err := rows.Scan(&id, &pid, &ptid); err != nil {
			return nil, fmt.Errorf("scan location id: %w", err)
		}
		out[LocationKey{PointID: pid, PointTypeID: ptid}] = id
	}
	return out, rows.Err()
}

// nullableFloat returns sql.NullFloat64{Valid:false} for NaN, otherwise Valid.
func nullableFloat(f float64) any {
	if f != f { // NaN guard - csvparse uses NaN for missing
		return nil
	}
	return f
}

// ForecastRow is a single forecast value bound for the forecasts table.
type ForecastRow struct {
	LocationID  int64
	ParameterID int64
	Timestamp   time.Time // UTC
	Value       float64
}

// UpsertForecasts inserts forecast rows; on conflict by (location, parameter,
// timestamp) the value is overwritten with the newer one. All rows are written
// in a single transaction with a prepared statement for throughput.
func UpsertForecasts(ctx context.Context, sqlDB *sql.DB, rows []ForecastRow) error {
	slog.Info("upserting forecasts", "count", len(rows))
	if len(rows) == 0 {
		return nil
	}

	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const upsertSQL = `
		INSERT INTO forecasts (timestamp, location_id, parameter_id, value)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (location_id, parameter_id, timestamp) DO UPDATE SET
			value = excluded.value
	`
	stmt, err := tx.PrepareContext(ctx, upsertSQL)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, r := range rows {
		ts := r.Timestamp.UTC().Format(time.RFC3339)
		if _, err := stmt.ExecContext(ctx, ts, r.LocationID, r.ParameterID, r.Value); err != nil {
			return fmt.Errorf("upsert forecast loc=%d param=%d ts=%s: %w",
				r.LocationID, r.ParameterID, ts, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	slog.Info("forecasts upserted", "count", len(rows))
	return nil
}
