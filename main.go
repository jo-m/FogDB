// Command nebeltracker runs one ingest cycle: download MeteoSwiss point
// forecast data (metadata + the configured parameter files) and write/upsert
// it into a local SQLite database.
//
// The intent is to schedule this binary to run hourly so the database
// accumulates a long-term archive of forecasts, where each (location,
// parameter, valid-time) tuple is overwritten by the most recent forecast.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	geo "github.com/kellydunn/golang-geo"

	"jo-m.ch/go/nebeltracker/internal/api"
	"jo-m.ch/go/nebeltracker/internal/csvparse"
	"jo-m.ch/go/nebeltracker/internal/db"
)

// wantedParameters are the forecast parameters we ingest. Extend the list to
// archive more variables.
var wantedParameters = []string{
	"jww003i0", // weather icon, 3h
	"rre150h0", // precipitation (mm), 1h
	"tre200h0", // air temperature 2m (degC), 1h
}

// Geographic filter: only forecast points within maxDistanceKm of the centre
// (Zürich) are stored in the database.
const (
	centreLat     = 47.371935
	centreLon     = 8.539336
	maxDistanceKm = 35.0
)

func main() {
	dbPath := flag.String("db", "nebeltracker.sqlite", "path to the SQLite database file")
	tmpDir := flag.String("tmp", "", "directory for staged CSV downloads (default: OS temp)")
	timeout := flag.Duration("timeout", 30*time.Minute, "overall workflow timeout")
	logLevel := flag.String("log-level", "info", "slog level: debug|info|warn|error")
	flag.Parse()

	setupLogger(*logLevel)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	ctx = withSignalCancel(ctx)

	if err := run(ctx, *dbPath, *tmpDir); err != nil {
		slog.Error("ingest workflow failed", "err", err)
		os.Exit(1)
	}
	slog.Info("ingest workflow finished successfully")
}

// run executes one ingest cycle.
func run(ctx context.Context, dbPath, tmpDir string) error {
	start := time.Now()

	stageDir, cleanup, err := makeStageDir(tmpDir)
	if err != nil {
		return fmt.Errorf("create stage dir: %w", err)
	}
	defer cleanup()
	slog.Info("staging directory ready", "path", stageDir)

	sqlDB, err := db.Open(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	client := api.NewClient()

	if err := syncMetadata(ctx, client, sqlDB, stageDir); err != nil {
		return fmt.Errorf("sync metadata: %w", err)
	}

	if err := ingestForecasts(ctx, client, sqlDB, stageDir); err != nil {
		return fmt.Errorf("ingest forecasts: %w", err)
	}

	slog.Info("workflow done", "elapsed", time.Since(start))
	return nil
}

// syncMetadata downloads the parameter and point metadata CSVs and upserts
// them into the database.
func syncMetadata(ctx context.Context, client *api.Client, sqlDB *sql.DB, stageDir string) error {
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

	filtered := filterLocationsByDistance(locs, centreLat, centreLon, maxDistanceKm)
	slog.Info("locations filtered by distance",
		"centre_lat", centreLat,
		"centre_lon", centreLon,
		"max_km", maxDistanceKm,
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

// makeStageDir prepares a directory for staged downloads. If parent is empty
// a fresh temporary directory is created; the cleanup func removes the whole
// tree. If parent is non-empty it is used as-is and never cleaned up.
func makeStageDir(parent string) (string, func(), error) {
	if parent == "" {
		dir, err := os.MkdirTemp("", "nebeltracker-*")
		if err != nil {
			return "", func() {}, err
		}
		return dir, func() {
			slog.Info("removing staging directory", "path", dir)
			_ = os.RemoveAll(dir)
		}, nil
	}
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", func() {}, err
	}
	return parent, func() {}, nil
}

func setupLogger(level string) {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:     lvl,
		AddSource: false,
	})
	slog.SetDefault(slog.New(handler))
}

// withSignalCancel returns ctx; it is cancelled on SIGINT/SIGTERM so the
// workflow stops cleanly mid-download.
func withSignalCancel(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case sig := <-sigCh:
			slog.Warn("received signal, cancelling", "signal", sig.String())
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx
}
