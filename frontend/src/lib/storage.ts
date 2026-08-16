/** localStorage key for the manually maintained location list. */
export const MANUAL_LOCATIONS_KEY = 'fogdb.manual-locations'

/**
 * Parse a persisted manual location list into an array of unique integer
 * point ids. Missing or malformed input yields an empty array.
 *
 * @param raw The raw string read from localStorage, or null.
 * @returns The parsed point ids, in first-seen order.
 */
export function parseManualPointIds(raw: string | null): number[] {
  if (raw == null) return []
  let parsed: unknown
  try {
    parsed = JSON.parse(raw)
  } catch {
    return []
  }
  if (!Array.isArray(parsed)) return []
  const seen = new Set<number>()
  const ids: number[] = []
  for (const value of parsed) {
    if (typeof value !== 'number' || !Number.isFinite(value)) continue
    const id = Math.trunc(value)
    if (seen.has(id)) continue
    seen.add(id)
    ids.push(id)
  }
  return ids
}

/**
 * Serialize a manual location list for persistence in localStorage.
 *
 * @param ids The point ids to persist.
 * @returns A JSON array string.
 */
export function serializeManualPointIds(ids: number[]): string {
  return JSON.stringify(ids)
}
