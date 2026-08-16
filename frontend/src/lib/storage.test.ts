import { describe, expect, it } from 'vitest'
import { parseManualPointIds, serializeManualPointIds } from './storage'

describe('parseManualPointIds', () => {
  it('returns an empty array for null or invalid input', () => {
    expect(parseManualPointIds(null)).toEqual([])
    expect(parseManualPointIds('not json')).toEqual([])
    expect(parseManualPointIds('"string"')).toEqual([])
    expect(parseManualPointIds('{}')).toEqual([])
  })

  it('parses integer ids and drops non-numbers', () => {
    expect(parseManualPointIds('[3, 1, 3, "4"]')).toEqual([3, 1])
    expect(parseManualPointIds('[1, null, 2]')).toEqual([1, 2])
  })

  it('truncates and deduplicates entries', () => {
    expect(parseManualPointIds('[3, 1, 3, 2.9]')).toEqual([3, 1, 2])
  })
})

describe('serializeManualPointIds', () => {
  it('serializes an array to JSON', () => {
    expect(serializeManualPointIds([1, 2, 3])).toBe('[1,2,3]')
  })
})
