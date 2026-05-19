# FogDB

One-shot Go binary: downloads MeteoSwiss point-forecast CSVs (STAC collection
`ch.meteoschweiz.ogd-local-forecasting`) and upserts them into a local
SQLite archive. Intended to be scheduled hourly. For each (location,
parameter) it keeps only the earliest-valid-time row of the run.

## Layout
- `main.go` - workflow: open/migrate DB, sync metadata, fetch latest assets, parse, filter by distance, collapse to earliest-per-location, upsert.
- `internal/api/` - STAC client + asset filename parsing (`vnut12.lssw.<YYYYMMDDHHmm>.<param>.csv`).
- `internal/csvparse/` - Latin1 (ISO-8859-1) semicolon CSV parsers for `meta_parameters`, `meta_point`, per-parameter forecasts. Test fixtures in `internal/csvparse/testdata/`.
- `internal/db/` - modernc.org/sqlite + embedded goose migrations (`migrations/*.sql`); `STRICT` tables.

## Schema
- `parameters` - unique `parameter_shortname`.
- `locations` - unique `(point_id, point_type_id)`; NaN coords stored as NULL.
- `forecasts` - unique `(location_id, parameter_id, timestamp)`, timestamp is RFC3339 UTC; upsert overwrites `value`. FK -> locations/parameters with `ON DELETE RESTRICT`. Indexed on `timestamp`.

## Run
`go run . [flags]`. All flags also accept the env var `FOGDB_<UPPER_SNAKE>`.
- `--db PATH` (default `db.sqlite`)
- `--log-level debug|info|warn|error` (default `info`)
- `--centre-lat`, `--centre-lon` (defaults: Zürich 47.371935, 8.539336)
- `--centre-max-distance-km` (default 35) - great-circle filter radius.

Workflow context timeout is hardcoded to 5 min in `main.main`.

## Hardcoded
- `wantedParameters` in `main.go` (`jww003i0`, `rre150h0`, `tre200h0`). Edit to ingest more.

## Dev
`make check` runs lint (gofmt, vet, staticcheck, revive, govulncheck, gosec) + build + tests. Must pass before declaring work done. Other targets: `make format`, `make test`, `make bench`.

## Conventions
- Public fns need docstrings.
- Code comments: complete sentences with terminal punctuation.
- Write tests for new code.
