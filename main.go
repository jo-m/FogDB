// Command fogdb periodically ingests MeteoSwiss point forecasts into a local
// SQLite database. On startup it runs one ingest cycle immediately, then
// repeats every --interval. Each cycle is bounded by --run-timeout via
// context cancellation, and SIGINT/SIGTERM cleanly stops the loop and any
// in-flight cycle.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	arg "github.com/alexflint/go-arg"

	"jo-m.ch/go/fogdb/internal/ingest"
)

// AppConfig contains application-wide configuration.
// It has struct tags compatible with [github.com/alexflint/go-arg].
//
//revive:disable:exported Naming necessary for struct embedding.
type AppConfig struct {
	// DBPath is the path to the SQLite database file.
	DBPath string `arg:"--db,env:FOGDB_DB" default:"db.sqlite" help:"Path to the SQLite database file" placeholder:"PATH"`
	// LogLevel is the log verbosity level passed to slog.
	LogLevel string `arg:"--log-level,env:FOGDB_LOG_LEVEL" default:"info" help:"Log level: debug|info|warn|error" placeholder:"LEVEL"`
	// CentreLat is the WGS84 latitude of the geographic filter centre.
	// Only forecast points within CentreMaxDistanceKm of (CentreLat, CentreLon) are stored.
	CentreLat float64 `arg:"--centre-lat,env:FOGDB_CENTRE_LAT" default:"47.371935" help:"WGS84 latitude of the geographic filter centre" placeholder:"LAT"`
	// CentreLon is the WGS84 longitude of the geographic filter centre.
	CentreLon float64 `arg:"--centre-lon,env:FOGDB_CENTRE_LON" default:"8.539336" help:"WGS84 longitude of the geographic filter centre" placeholder:"LON"`
	// CentreMaxDistanceKm is the radius of the geographic filter in kilometres.
	CentreMaxDistanceKm float64 `arg:"--centre-max-distance-km,env:FOGDB_CENTRE_MAX_DISTANCE_KM" default:"35" help:"Radius of the geographic filter in kilometres" placeholder:"KM"`
	// Interval is the wall-clock spacing between successive ingest cycles.
	Interval time.Duration `arg:"--interval,env:FOGDB_INTERVAL" default:"45m" help:"Interval between ingest cycles" placeholder:"DUR"`
	// RunTimeout bounds the duration of a single ingest cycle.
	RunTimeout time.Duration `arg:"--run-timeout,env:FOGDB_RUN_TIMEOUT" default:"5m" help:"Timeout applied to each ingest cycle" placeholder:"DUR"`
}

func main() {
	var cfg AppConfig
	arg.MustParse(&cfg)

	setupLogger(cfg.LogLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = withSignalCancel(ctx)

	ingestCfg := ingest.Config{
		DBPath:              cfg.DBPath,
		CentreLat:           cfg.CentreLat,
		CentreLon:           cfg.CentreLon,
		CentreMaxDistanceKm: cfg.CentreMaxDistanceKm,
	}

	slog.Info("starting ingest loop",
		"interval", cfg.Interval,
		"run_timeout", cfg.RunTimeout,
	)

	// Run once immediately so the database is populated without waiting a
	// full interval after startup.
	runOnce(ctx, ingestCfg, cfg.RunTimeout)

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("shutting down", "reason", ctx.Err())
			return
		case <-ticker.C:
			runOnce(ctx, ingestCfg, cfg.RunTimeout)
		}
	}
}

// runOnce executes a single ingest cycle bounded by timeout. Errors are logged
// but not returned: the caller is the ticker loop, which must keep running on
// transient failures.
func runOnce(parent context.Context, cfg ingest.Config, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	if err := ingest.Run(ctx, cfg); err != nil {
		slog.Error("ingest cycle failed", "err", err)
		return
	}
	slog.Info("ingest cycle finished successfully")
}

// setupLogger installs a slog text handler on stderr at the requested level.
// An unparseable level falls back to info.
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
