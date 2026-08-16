import SqlJs from 'sql.js'
import wasmUrl from 'sql.js/dist/sql-wasm.wasm?url'

type Database = SqlJs.Database
type SqlValue = SqlJs.SqlValue
type BindParams = SqlJs.BindParams

/** A forecast parameter that has at least one data row in the database. */
export interface Parameter {
  id: number
  shortname: string
  description: string
  unit: string
  decimals: number
}

/** A location that has at least one forecast row in the database. */
export interface Location {
  id: number
  pointId: number
  name: string
  abbr: string
  lat: number
  lon: number
  height: number | null
}

/** A single forecast value joined with its location, for one timestamp. */
export interface LocationValue {
  locationId: number
  name: string
  abbr: string
  lat: number
  lon: number
  value: number
}

/** Everything loaded from an uploaded database file. */
export interface LoadedDb {
  db: Database
  parameters: Parameter[]
  locations: Location[]
  locationById: Map<number, Location>
}

let sqlJsPromise: Promise<SqlJs.SqlJsStatic> | null = null

/**
 * Lazily initialize the sql.js WebAssembly module. The result is cached so the
 * (expensive) initialization happens at most once per page load.
 *
 * @returns The initialized sql.js module.
 */
function getSqlJs(): Promise<SqlJs.SqlJsStatic> {
  if (!sqlJsPromise) {
    sqlJsPromise = SqlJs({ locateFile: () => wasmUrl })
  }
  return sqlJsPromise
}

/** Coerce a sql.js cell value to a number. */
function asNumber(v: SqlValue): number {
  return typeof v === 'number' ? v : Number(v)
}

/** Coerce a sql.js cell value to a string. */
function asString(v: SqlValue): string {
  return v == null ? '' : String(v)
}

/**
 * Execute a query and return all rows as plain objects keyed by column name.
 *
 * @param db The open database.
 * @param sql A single SQL statement, optionally with `?` placeholders.
 * @param params Positional bind parameters.
 * @returns One object per result row.
 */
export function queryAll(
  db: Database,
  sql: string,
  params: BindParams = []
): Record<string, SqlValue>[] {
  const result = db.exec(sql, params)
  if (result.length === 0) return []
  const { columns, values } = result[0]
  return values.map((row) => {
    const obj: Record<string, SqlValue> = {}
    for (let i = 0; i < columns.length; i++) {
      obj[columns[i]] = row[i]
    }
    return obj
  })
}

/**
 * Open an uploaded SQLite database file and extract the parameters and
 * locations that have at least one forecast row.
 *
 * @param file The uploaded file.
 * @returns The open database plus its parameters and locations.
 */
export async function loadDatabase(file: File): Promise<LoadedDb> {
  const SQL = await getSqlJs()
  const buffer = await file.arrayBuffer()
  const db = new SQL.Database(new Uint8Array(buffer))

  const parameterRows = queryAll(
    db,
    `SELECT id, parameter_shortname, parameter_description_en,
            parameter_unit, parameter_decimals
     FROM parameters p
     WHERE EXISTS (SELECT 1 FROM forecasts f WHERE f.parameter_id = p.id)
     ORDER BY p.parameter_shortname`
  )

  const parameters: Parameter[] = parameterRows.map((r) => ({
    id: asNumber(r.id),
    shortname: asString(r.parameter_shortname),
    description: asString(r.parameter_description_en),
    unit: asString(r.parameter_unit),
    decimals: asNumber(r.parameter_decimals)
  }))

  const locationRows = queryAll(
    db,
    `SELECT id, point_id, point_name, station_abbr,
            point_height_masl,
            point_coordinates_wgs84_lat AS lat,
            point_coordinates_wgs84_lon AS lon
     FROM locations l
     WHERE EXISTS (SELECT 1 FROM forecasts f WHERE f.location_id = l.id)
     ORDER BY l.point_name`
  )

  const locations: Location[] = locationRows.map((r) => ({
    id: asNumber(r.id),
    pointId: asNumber(r.point_id),
    name: asString(r.point_name),
    abbr: asString(r.station_abbr),
    height: r.point_height_masl == null ? null : asNumber(r.point_height_masl),
    lat: asNumber(r.lat),
    lon: asNumber(r.lon)
  }))

  const locationById = new Map(locations.map((l) => [l.id, l]))

  return { db, parameters, locations, locationById }
}

/**
 * Return the distinct forecast timestamps of a parameter, ascending.
 *
 * @param db The open database.
 * @param parameterId The parameter primary key.
 * @returns RFC3339 UTC timestamp strings.
 */
export function getTimestamps(db: Database, parameterId: number): string[] {
  return queryAll(
    db,
    'SELECT DISTINCT timestamp FROM forecasts WHERE parameter_id = ? ORDER BY timestamp',
    [parameterId]
  ).map((r) => asString(r.timestamp))
}

/**
 * Return the joined location/value rows for one parameter and timestamp.
 *
 * @param db The open database.
 * @param parameterId The parameter primary key.
 * @param timestamp The RFC3339 UTC timestamp.
 * @param locationById Map from location id to location metadata.
 * @returns One row per location that has a value at the timestamp.
 */
export function getValues(
  db: Database,
  parameterId: number,
  timestamp: string,
  locationById: Map<number, Location>
): LocationValue[] {
  const rows = queryAll(
    db,
    'SELECT location_id, value FROM forecasts WHERE parameter_id = ? AND timestamp = ?',
    [parameterId, timestamp]
  )
  return rows
    .map((r) => {
      const location = locationById.get(asNumber(r.location_id))
      if (!location) return null
      return {
        locationId: location.id,
        name: location.name,
        abbr: location.abbr,
        lat: location.lat,
        lon: location.lon,
        value: asNumber(r.value)
      }
    })
    .filter((v): v is LocationValue => v !== null)
}
