# FogDB

Long-running Go binary that ingests MeteoSwiss point-forecast CSVs (STAC collection `ch.meteoschweiz.ogd-local-forecasting`) into a local SQLite archive.
Runs one cycle on startup, then repeats on a configurable interval, keeping only the earliest-valid-time row per (location, parameter) per run.

## Run

```
go run . [flags]
```

Flags (env var equivalents: `FOGDB_<UPPER_SNAKE>`):

- `--db PATH` (default `db.sqlite`)
- `--log-level debug|info|warn|error` (default `info`)
- `--centre-lat`, `--centre-lon` (default: Zürich)
- `--centre-max-distance-km` (default `35`)
- `--interval DUR` (default `45m`)
- `--run-timeout DUR` (default `5m`)

## Dev

`make check` runs lint, build, and tests. See `CLAUDE.md` for layout and schema details.
