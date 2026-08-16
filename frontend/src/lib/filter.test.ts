import { describe, expect, it } from 'vitest'
import type { Location } from './db'
import { filterByPointIds, filterByViewport, inBbox } from './filter'

function location(id: number, pointId: number, lat: number, lon: number): Location {
  return { id, pointId, name: 'x', abbr: '', lat, lon, height: null }
}

describe('inBbox', () => {
  const bbox = { minLat: 46, minLon: 6, maxLat: 48, maxLon: 10 }

  it('accepts points inside the box', () => {
    expect(inBbox(bbox, 47, 8)).toBe(true)
  })

  it('accepts points on the boundary', () => {
    expect(inBbox(bbox, 46, 10)).toBe(true)
  })

  it('rejects points outside the box', () => {
    expect(inBbox(bbox, 45, 8)).toBe(false)
    expect(inBbox(bbox, 47, 11)).toBe(false)
  })
})

describe('filterByViewport', () => {
  const locations = [
    location(1, 100, 47.3, 8.5),
    location(2, 101, 45.9, 7.0),
    location(3, 102, 46.2, 8.0)
  ]

  it('keeps only locations inside the bounding box', () => {
    const bbox = { minLat: 46, minLon: 6, maxLat: 48, maxLon: 9 }
    expect(filterByViewport(locations, bbox).map((l) => l.id)).toEqual([1, 3])
  })
})

describe('filterByPointIds', () => {
  const locations = [
    location(1, 100, 47.3, 8.5),
    location(2, 101, 45.9, 7.0),
    location(3, 102, 46.2, 9.5)
  ]

  it('keeps only locations whose point id is allowed', () => {
    expect(filterByPointIds(locations, new Set([101, 102])).map((l) => l.id)).toEqual([2, 3])
  })
})
