import type { Location } from './db'

/**
 * Normalize text for matching: lowercase it and strip combining marks so
 * names such as `Zürich` and `zurich` compare equal.
 *
 * @param text The raw text.
 * @returns The normalized text.
 */
export function normalizeForSearch(text: string): string {
  return text
    .toLowerCase()
    .normalize('NFD')
    .replace(/\p{M}/gu, '')
    .trim()
}

/**
 * Score how well a query matches a target for fuzzy search. Lower is better;
 * a negative result means no match.
 *
 * @param query The user query.
 * @param target The text being matched against.
 * @returns A non-negative score, or -1 when there is no match.
 */
export function fuzzyScore(query: string, target: string): number {
  const q = normalizeForSearch(query)
  const t = normalizeForSearch(target)
  if (q.length === 0) return -1
  if (t === q) return 0
  if (t.startsWith(q)) return 1
  const index = t.indexOf(q)
  if (index >= 0) return 2 + index
  // Fall back to a subsequence match: all query characters must appear in
  // order. Gaps between matched characters are penalized.
  let ti = 0
  let score = 0
  for (let qi = 0; qi < q.length; qi++) {
    const found = t.indexOf(q[qi], ti)
    if (found < 0) return -1
    score += found - ti + 1
    ti = found + 1
  }
  return 10 + score
}

/**
 * Rank locations by how well their name or abbreviation matches a query.
 *
 * @param locations Candidate locations.
 * @param query The user query.
 * @param limit Maximum number of results to return.
 * @returns The best matches, ordered by score ascending.
 */
export function searchLocations(locations: Location[], query: string, limit = 10): Location[] {
  if (normalizeForSearch(query).length === 0) return []
  return locations
    .map((location) => {
      const nameScore = fuzzyScore(query, location.name)
      const abbrScore = fuzzyScore(query, location.abbr)
      const score = Math.min(
        nameScore < 0 ? Infinity : nameScore,
        abbrScore < 0 ? Infinity : abbrScore
      )
      return { location, score }
    })
    .filter((entry) => Number.isFinite(entry.score))
    .sort((a, b) => a.score - b.score)
    .slice(0, limit)
    .map((entry) => entry.location)
}
