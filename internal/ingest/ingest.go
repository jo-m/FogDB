// Package ingest implements one MeteoSwiss point-forecast ingest cycle:
// download metadata + the configured parameter files and upsert them into the
// local SQLite database.
package ingest

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"time"

	geo "github.com/kellydunn/golang-geo"

	"jo-m.ch/go/fogdb/internal/api"
	"jo-m.ch/go/fogdb/internal/csvparse"
	"jo-m.ch/go/fogdb/internal/db"
)

// wantedParameters are the forecast parameters we ingest. Extend the list to
// archive more variables.
var wantedParameters = []string{
	"jww003i0", // weather icon, 3h
	"rre150h0", // precipitation (mm), 1h
	"tre200h0", // air temperature 2m (degC), 1h
}

// Config bundles the parameters that govern a single ingest cycle.
type Config struct {
	// DBPath is the path to the SQLite database file.
	DBPath string
	// CentreLat is the WGS84 latitude of the geographic filter centre.
	CentreLat float64
	// CentreLon is the WGS84 longitude of the geographic filter centre.
	CentreLon float64
	// CentreMaxDistanceKm is the radius (km) of the great-circle filter
	// centred on (CentreLat, CentreLon). Locations farther than this are
	// dropped before being stored.
	CentreMaxDistanceKm float64
}

// Run executes one ingest cycle: open/migrate the database, sync parameter
// and location metadata, fetch the latest forecast asset for each wanted
// parameter, collapse to the earliest-valid-time row per location, and upsert
// the result. The cycle aborts as soon as ctx is cancelled.
func Run(ctx context.Context, cfg Config) error {
	start := time.Now()

	stageDir, cleanup, err := makeStageDir()
	if err != nil {
		return fmt.Errorf("create stage dir: %w", err)
	}
	defer cleanup()
	slog.Info("staging directory ready", "path", stageDir)

	sqlDB, err := db.Open(ctx, cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	client := api.NewClient()

	if err := syncMetadata(ctx, client, sqlDB, stageDir, cfg.CentreLat, cfg.CentreLon, cfg.CentreMaxDistanceKm); err != nil {
		return fmt.Errorf("sync metadata: %w", err)
	}

	if err := ingestForecasts(ctx, client, sqlDB, stageDir); err != nil {
		return fmt.Errorf("ingest forecasts: %w", err)
	}

	slog.Info("workflow done", "elapsed", time.Since(start))
	return nil
}

// syncMetadata downloads the parameter and point metadata CSVs and upserts
// them into the database. Only locations within maxKm of (centreLat, centreLon)
// are stored.
func syncMetadata(ctx context.Context, client *api.Client, sqlDB *sql.DB, stageDir string, centreLat, centreLon, maxKm float64) error {
	paramsPath := filepath.Join(stageDir, "meta_parameters.csv")
	pointsPath := filepath.Join(stageDir, "meta_point.csv")

	if err := downloadToFile(ctx, client, client.MetaParametersURL(), paramsPath); err != nil {
		return fmt.Errorf("download meta_parameters: %w", err)
	}
	if err := downloadToFile(ctx, client, client.MetaPointsURL(), pointsPath); err != nil {
		return fmt.Errorf("download meta_point: %w", err)
	}

	paramsFile, err := os.Open(paramsPath)
	if err != nil {
		return fmt.Errorf("open meta_parameters: %w", err)
	}
	defer func() { _ = paramsFile.Close() }()
	params, err := csvparse.ParseParameters(paramsFile)
	if err != nil {
		return fmt.Errorf("parse meta_parameters: %w", err)
	}
	slog.Info("parsed meta_parameters", "count", len(params))

	pointsFile, err := os.Open(pointsPath)
	if err != nil {
		return fmt.Errorf("open meta_point: %w", err)
	}
	defer func() { _ = pointsFile.Close() }()
	locs, err := csvparse.ParseLocations(pointsFile)
	if err != nil {
		return fmt.Errorf("parse meta_point: %w", err)
	}
	slog.Info("parsed meta_point", "count", len(locs))

	filtered := filterLocationsByDistance(locs, centreLat, centreLon, maxKm)
	slog.Info("locations filtered by distance",
		"centre_lat", centreLat,
		"centre_lon", centreLon,
		"max_km", maxKm,
		"kept", len(filtered),
		"dropped", len(locs)-len(filtered),
	)

	if _, err := db.UpsertParameters(ctx, sqlDB, params); err != nil {
		return fmt.Errorf("upsert parameters: %w", err)
	}
	if _, err := db.UpsertLocations(ctx, sqlDB, filtered); err != nil {
		return fmt.Errorf("upsert locations: %w", err)
	}
	return nil
}

// filterLocationsByDistance returns the subset of locs within maxKm of
// (lat,lon), measured as the great-circle distance. Locations with missing
// WGS84 coordinates are dropped.
func filterLocationsByDistance(locs []csvparse.Location, lat, lon, maxKm float64) []csvparse.Location {
	centre := geo.NewPoint(lat, lon)
	out := make([]csvparse.Location, 0, len(locs))
	for _, l := range locs {
		if math.IsNaN(l.WGS84Lat) || math.IsNaN(l.WGS84Lon) {
			continue
		}
		p := geo.NewPoint(l.WGS84Lat, l.WGS84Lon)
		if centre.GreatCircleDistance(p) <= maxKm {
			out = append(out, l)
		}
	}
	return out
}

// ingestForecasts discovers and downloads the latest forecast asset for each
// wanted parameter, parses the CSVs, and upserts forecast rows.
func ingestForecasts(ctx context.Context, client *api.Client, sqlDB *sql.DB, stageDir string) error {
	assets, err := client.LatestAssets(ctx, wantedParameters)
	if err != nil {
		return fmt.Errorf("find latest assets: %w", err)
	}
	for p, a := range assets {
		slog.Info("selected forecast asset",
			"parameter", p,
			"run_time_utc", a.RunTime.Format(time.RFC3339),
			"feature", a.FeatureID,
			"filename", a.Filename,
		)
	}

	// Load id maps once; they cover every (point_id, point_type_id) and
	// shortname present in the metadata sync above.
	paramIDs, err := loadParameterIDs(ctx, sqlDB)
	if err != nil {
		return err
	}
	locIDs, err := loadLocationIDs(ctx, sqlDB)
	if err != nil {
		return err
	}

	for _, param := range wantedParameters {
		a := assets[param]
		path := filepath.Join(stageDir, a.Filename)
		if err := downloadToFile(ctx, client, a.Href, path); err != nil {
			return fmt.Errorf("download %s: %w", a.Filename, err)
		}

		rows, err := parseForecastFile(path)
		if err != nil {
			return fmt.Errorf("parse %s: %w", a.Filename, err)
		}
		slog.Info("parsed forecast file",
			"parameter", param,
			"rows", len(rows),
		)

		paramID, ok := paramIDs[param]
		if !ok {
			return fmt.Errorf("parameter %q has no id in db (metadata sync missing?)", param)
		}

		dbRows, skipped, err := buildForecastRows(rows, paramID, locIDs)
		if err != nil {
			return fmt.Errorf("build db rows for %s: %w", param, err)
		}
		if skipped > 0 {
			// Most rows belong to points outside the configured distance
			// filter and are intentionally dropped here.
			slog.Info("dropped forecast rows for non-stored locations",
				"parameter", param,
				"dropped", skipped,
				"kept", len(dbRows),
			)
		}

		// For archival we keep only the earliest-timestamp row per location
		// (the "now" forecast of this run); the rest of the horizon is
		// discarded so the database grows linearly in time, not in horizon.
		before := len(dbRows)
		dbRows = keepEarliestPerLocation(dbRows)
		slog.Info("collapsed to earliest-per-location",
			"parameter", param,
			"input_rows", before,
			"kept_rows", len(dbRows),
		)

		if err := db.UpsertForecasts(ctx, sqlDB, dbRows); err != nil {
			return fmt.Errorf("upsert forecasts for %s: %w", param, err)
		}
	}
	return nil
}

// buildForecastRows joins parsed forecast rows with internal location IDs.
// Forecasts referencing a location not present in the metadata are skipped
// (this shouldn't normally happen).
func buildForecastRows(
	rows []csvparse.ForecastRow,
	paramID int64,
	locIDs map[db.LocationKey]int64,
) (out []db.ForecastRow, skipped int, err error) {
	out = make([]db.ForecastRow, 0, len(rows))
	for _, r := range rows {
		key := db.LocationKey{PointID: r.PointID, PointTypeID: r.PointTypeID}
		locID, ok := locIDs[key]
		if !ok {
			skipped++
			continue
		}
		out = append(out, db.ForecastRow{
			LocationID:  locID,
			ParameterID: paramID,
			Timestamp:   r.ValidTime,
			Value:       r.Value,
		})
	}
	return out, skipped, nil
}

// keepEarliestPerLocation reduces rows to one entry per LocationID, namely
// the one with the smallest Timestamp. All input rows are assumed to share
// the same ParameterID (this is called once per parameter). Output order is
// not specified.
func keepEarliestPerLocation(rows []db.ForecastRow) []db.ForecastRow {
	best := make(map[int64]db.ForecastRow, len(rows))
	for _, r := range rows {
		cur, ok := best[r.LocationID]
		if !ok || r.Timestamp.Before(cur.Timestamp) {
			best[r.LocationID] = r
		}
	}
	out := make([]db.ForecastRow, 0, len(best))
	for _, r := range best {
		out = append(out, r)
	}
	return out
}

// parseForecastFile is a small helper that opens & parses a forecast CSV.
func parseForecastFile(path string) ([]csvparse.ForecastRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer func() { _ = f.Close() }()
	return csvparse.ParseForecast(f)
}

// loadParameterIDs queries the parameters table; mirror of db.loadParameterIDs
// but exposed here because the latter is private.
func loadParameterIDs(ctx context.Context, sqlDB *sql.DB) (map[string]int64, error) {
	rows, err := sqlDB.QueryContext(ctx, `SELECT id, parameter_shortname FROM parameters`)
	if err != nil {
		return nil, fmt.Errorf("query parameters: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make(map[string]int64)
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out[name] = id
	}
	return out, rows.Err()
}

// loadLocationIDs queries the locations table and returns a map from
// (point_id, point_type_id) to the internal location id.
func loadLocationIDs(ctx context.Context, sqlDB *sql.DB) (map[db.LocationKey]int64, error) {
	rows, err := sqlDB.QueryContext(ctx, `SELECT id, point_id, point_type_id FROM locations`)
	if err != nil {
		return nil, fmt.Errorf("query locations: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make(map[db.LocationKey]int64)
	for rows.Next() {
		var id, pid, ptid int64
		if err := rows.Scan(&id, &pid, &ptid); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out[db.LocationKey{PointID: pid, PointTypeID: ptid}] = id
	}
	return out, rows.Err()
}

// downloadToFile streams a URL into a file at path (overwriting any existing).
func downloadToFile(ctx context.Context, client *api.Client, url, path string) error {
	tmp := path + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create %s: %w", tmp, err)
	}
	n, err := client.Download(ctx, url, f)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", tmp, path, err)
	}
	slog.Info("staged file", "path", path, "bytes", n)
	return nil
}

// makeStageDir creates a fresh OS temporary directory for staged downloads.
// The returned cleanup func removes the whole tree.
func makeStageDir() (string, func(), error) {
	dir, err := os.MkdirTemp("", "fogdb-*")
	if err != nil {
		return "", func() {}, err
	}
	return dir, func() {
		slog.Info("removing staging directory", "path", dir)
		_ = os.RemoveAll(dir)
	}, nil
}
