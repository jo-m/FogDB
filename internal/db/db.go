package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // sqlite driver registration
)

// pragmaDSN converts a plain file path into a modernc.org/sqlite DSN that
// encodes all desired PRAGMAs as _pragma query parameters. This ensures every
// connection opened by the sql.DB pool inherits the settings, not just the
// first one.
func pragmaDSN(path string) string {
	q := url.Values{}
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "synchronous(NORMAL)")
	q.Add("_pragma", "temp_store(MEMORY)")
	q.Add("_pragma", "cache_size(1000000000)")
	q.Add("_pragma", "foreign_keys(ON)")
	q.Add("_pragma", "mmap_size(2147483648)")
	return "file:" + path + "?" + q.Encode()
}

// Open opens (or creates) a SQLite database at path and applies all pending
// goose migrations. PRAGMAs are embedded in the DSN so they are applied on
// every new connection in the pool.
//
// path is a plain filesystem path (e.g. "nebeltracker.sqlite").
func Open(ctx context.Context, path string) (*sql.DB, error) {
	dsn := pragmaDSN(path)
	slog.Info("opening sqlite database", "path", path)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	if err := migrate(ctx, sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return sqlDB, nil
}

// migrate applies all embedded goose migrations.
func migrate(ctx context.Context, sqlDB *sql.DB) error {
	migrationsSub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("locate embedded migrations: %w", err)
	}
	provider, err := goose.NewProvider(
		goose.DialectSQLite3,
		sqlDB,
		migrationsSub,
		goose.WithVerbose(false),
	)
	if err != nil {
		return fmt.Errorf("goose.NewProvider: %w", err)
	}
	slog.Info("applying database migrations")
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("goose.Up: %w", err)
	}
	for _, r := range results {
		slog.Info("migration applied",
			"version", r.Source.Version,
			"name", r.Source.Path,
			"duration", r.Duration,
		)
	}
	if len(results) == 0 {
		slog.Info("database schema already up to date")
	}
	return nil
}
