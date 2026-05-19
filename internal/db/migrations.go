// Package db handles the SQLite store: opening, migrating, and upserting
// MeteoSwiss point-forecast data.
package db

import "embed"

//go:embed migrations/*.sql
var migrationsFS embed.FS
