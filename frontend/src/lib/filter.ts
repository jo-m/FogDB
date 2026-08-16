import type { Location } from './db'

/** Axis-aligned geographic bounding box in WGS84 degrees. */
export interface WgsBbox {
  minLat: number
  minLon: number
  maxLat: number
  maxLon: number
}

/**
 * Return whether a WGS84 coordinate lies within a bounding box. The box is
 * inclusive on all four edges.
 *
 * @param bbox The bounding box.
 * @param lat Latitude in degrees.
 * @param lon Longitude in degrees.
 * @returns True when the coordinate falls inside the box.
 */
export function inBbox(bbox: WgsBbox, lat: number, lon: number): boolean {
  return lon >= bbox.minLon && lon <= bbox.maxLon && lat >= bbox.minLat && lat <= bbox.maxLat
}

/**
 * Keep only the locations whose coordinates fall inside a bounding box.
 *
 * @param locations Candidate locations.
 * @param bbox The bounding box.
 * @returns The subset of locations inside the box.
 */
export function filterByViewport(locations: Location[], bbox: WgsBbox): Location[] {
  return locations.filter((l) => inBbox(bbox, l.lat, l.lon))
}

/**
 * Keep only the locations whose point id is present in a set.
 *
 * @param locations Candidate locations.
 * @param pointIds The set of allowed point ids.
 * @returns The subset of locations whose point id is allowed.
 */
export function filterByPointIds(locations: Location[], pointIds: ReadonlySet<number>): Location[] {
  return locations.filter((l) => pointIds.has(l.pointId))
}
