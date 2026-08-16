import { describe, expect, it } from 'vitest'
import type { Location } from './db'
import { fuzzyScore, normalizeForSearch, searchLocations } from './search'

function location(id: number, pointId: number, name: string, abbr = ''): Location {
  return { id, pointId, name, abbr, lat: 0, lon: 0, height: null }
}

describe('normalizeForSearch', () => {
  it('lowercases, trims and strips diacritics', () => {
    expect(normalizeForSearch('  Zürich ')).toBe('zurich')
    expect(normalizeForSearch('Genève')).toBe('geneve')
  })
})

describe('fuzzyScore', () => {
  it('scores an exact match best', () => {
    expect(fuzzyScore('zurich', 'zurich')).toBe(0)
  })

  it('scores a prefix match next', () => {
    expect(fuzzyScore('zur', 'zurich')).toBe(1)
  })

  it('scores a substring match by position', () => {
    expect(fuzzyScore('uri', 'zurich')).toBe(3)
  })

  it('scores a subsequence match worst', () => {
    expect(fuzzyScore('zch', 'zurich')).toBeGreaterThan(10)
  })

  it('returns -1 for no match', () => {
    expect(fuzzyScore('xyz', 'zurich')).toBe(-1)
  })

  it('is case and diacritic insensitive', () => {
    expect(fuzzyScore('ZÜRICH', 'zurich')).toBe(0)
  })
})

describe('searchLocations', () => {
  const locations = [
    location(1, 100, 'Zurich', 'ZRH'),
    location(2, 101, 'Bern', 'BER'),
    location(3, 102, 'Geneva', 'GVA'),
    location(4, 103, 'Zurich Airport', '')
  ]

  it('returns an empty list for a blank query', () => {
    expect(searchLocations(locations, '   ')).toEqual([])
  })

  it('ranks better matches first and respects the limit', () => {
    expect(searchLocations(locations, 'zur', 2).map((l) => l.name)).toEqual([
      'Zurich',
      'Zurich Airport'
    ])
  })

  it('matches by abbreviation', () => {
    expect(searchLocations(locations, 'gva').map((l) => l.name)).toEqual(['Geneva'])
  })
})
