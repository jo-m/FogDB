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
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	arg "github.com/alexflint/go-arg"
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

// AppConfig contains application-wide configuration.
// It has struct tags compatible with [github.com/alexflint/go-arg].
//
//revive:disable:exported Naming necessary for struct embedding.
type AppConfig struct {
	// DBPath is the path to the SQLite database file.
	DBPath string `arg:"--db,env:NEBELTRACKER_DB" default:"nebeltracker.sqlite" help:"Path to the SQLite database file" placeholder:"PATH"`
	// LogLevel is the log verbosity level passed to slog.
	LogLevel string `arg:"--log-level,env:NEBELTRACKER_LOG_LEVEL" default:"info" help:"Log level: debug|info|warn|error" placeholder:"LEVEL"`
	// CentreLat is the WGS84 latitude of the geographic filter centre.
	// Only forecast points within CentreMaxDistanceKm of (CentreLat, CentreLon) are stored.
	CentreLat float64 `arg:"--centre-lat,env:NEBELTRACKER_CENTRE_LAT" default:"47.371935" help:"WGS84 latitude of the geographic filter centre" placeholder:"LAT"`
	// CentreLon is the WGS84 longitude of the geographic filter centre.
	CentreLon float64 `arg:"--centre-lon,env:NEBELTRACKER_CENTRE_LON" default:"8.539336" help:"WGS84 longitude of the geographic filter centre" placeholder:"LON"`
	// CentreMaxDistanceKm is the radius of the geographic filter in kilometres.
	CentreMaxDistanceKm float64 `arg:"--centre-max-distance-km,env:NEBELTRACKER_CENTRE_MAX_DISTANCE_KM" default:"35" help:"Radius of the geographic filter in kilometres" placeholder:"KM"`
}

func main() {
	var cfg AppConfig
	arg.MustParse(&cfg)

	setupLogger(cfg.LogLevel)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctx = withSignalCancel(ctx)

	if err := run(ctx, cfg); err != nil {
		slog.Error("ingest workflow failed", "err", err)
		os.Exit(1)
	}
	slog.Info("ingest workflow finished successfully")
}

// run executes one ingest cycle.
func run(ctx context.Context, cfg AppConfig) error {
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
	dir, err := os.MkdirTemp("", "nebeltracker-*")
	if err != nil {
		return "", func() {}, err
	}
	return dir, func() {
		slog.Info("removing staging directory", "path", dir)
		_ = os.RemoveAll(dir)
	}, nil
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
