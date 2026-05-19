# nebeltracker

Hourly ingest of MeteoSwiss point-forecast CSVs into a local SQLite archive.
Records only the earliest-valid-time row per (location, parameter) per run,
restricted to points within `maxDistanceKm` of Zürich.

## Layout
- `main.go` - workflow: open/migrate DB, sync metadata, fetch latest assets, parse, filter, upsert.
- `internal/db/` - modernc sqlite + embedded goose migrations (`migrations/*.sql`).
- `internal/csvparse/` - Latin1 CSV parsers for meta_parameters, meta_point, per-parameter forecast files. Test fixtures in `internal/csvparse/testdata/`.
- `internal/api/` - STAC client for `ch.meteoschweiz.ogd-local-forecasting`.

## Schema
- `parameters` (unique `parameter_shortname`)
- `locations` (unique `(point_id, point_type_id)`)
- `forecasts` (unique `(location_id, parameter_id, timestamp)`, RFC3339 UTC); upsert overwrites value.

## Run
`go run . -db nebeltracker.sqlite` (flags: `-tmp`, `-timeout`, `-log-level`)

## Dev tools
`make check` runs lint (gofmt, vet, staticcheck, revive, govulncheck, gosec) + build + tests. Must pass before declaring work done. Other targets: `make format`, `make test`, `make bench`.

## Conventions
- All public fns need docstrings.
- Code comments: grammatical complete sentences ending in punctuation.
- MUST write tests for all new code.
