package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // sqlite driver registration
)

// pragmas lists SQLite PRAGMA settings applied on every connection open.
var pragmas = []struct{ name, value string }{
	{"journal_mode", "WAL"},
	{"synchronous", "NORMAL"},
	{"temp_store", "MEMORY"},
	{"cache_size", "1000000000"},
	{"foreign_keys", "ON"},
	{"mmap_size", "2147483648"},
}

// Open opens (or creates) a SQLite database at dsn and applies all pending
// goose migrations. The pragmas in [pragmas] are set before running migrations.
//
// dsn is a modernc.org/sqlite DSN, typically a file path or
// "file:foo.db?_pragma=...".
func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	slog.Info("opening sqlite database", "dsn", dsn)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	for _, p := range pragmas {
		if _, err := sqlDB.ExecContext(ctx, "PRAGMA "+p.name+" = "+p.value+";"); err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("set pragma %s: %w", p.name, err)
		}
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
