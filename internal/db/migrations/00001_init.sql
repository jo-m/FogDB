-- +goose Up
-- +goose StatementBegin
CREATE TABLE parameters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    parameter_shortname TEXT NOT NULL UNIQUE,
    parameter_description_de TEXT NOT NULL,
    parameter_description_fr TEXT NOT NULL,
    parameter_description_it TEXT NOT NULL,
    parameter_description_en TEXT NOT NULL,
    parameter_group_de TEXT NOT NULL,
    parameter_group_fr TEXT NOT NULL,
    parameter_group_it TEXT NOT NULL,
    parameter_group_en TEXT NOT NULL,
    parameter_granularity TEXT NOT NULL,
    parameter_decimals INTEGER NOT NULL,
    parameter_datatype TEXT NOT NULL,
    parameter_unit TEXT NOT NULL
) STRICT;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE locations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    point_id INTEGER NOT NULL,
    point_type_id INTEGER NOT NULL,
    station_abbr TEXT NOT NULL DEFAULT '',
    postal_code TEXT NOT NULL DEFAULT '',
    point_name TEXT NOT NULL,
    point_type_de TEXT NOT NULL,
    point_type_fr TEXT NOT NULL,
    point_type_it TEXT NOT NULL,
    point_type_en TEXT NOT NULL,
    point_height_masl REAL,
    point_coordinates_lv95_east REAL,
    point_coordinates_lv95_north REAL,
    point_coordinates_wgs84_lat REAL,
    point_coordinates_wgs84_lon REAL,
    UNIQUE (point_id, point_type_id)
) STRICT;
-- +goose StatementEnd

-- +goose StatementBegin
-- Forecast values keyed by (location, parameter, valid timestamp).
-- timestamp is stored as RFC3339 UTC, e.g. "2026-05-19T04:00:00Z".
CREATE TABLE forecasts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp TEXT NOT NULL,
    location_id INTEGER NOT NULL REFERENCES locations(id) ON DELETE RESTRICT,
    parameter_id INTEGER NOT NULL REFERENCES parameters(id) ON DELETE RESTRICT,
    value REAL NOT NULL,
    UNIQUE (location_id, parameter_id, timestamp)
) STRICT;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_forecasts_timestamp ON forecasts(timestamp);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_forecasts_timestamp;
DROP TABLE IF EXISTS forecasts;
DROP TABLE IF EXISTS locations;
DROP TABLE IF EXISTS parameters;
-- +goose StatementEnd
