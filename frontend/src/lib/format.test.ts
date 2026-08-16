import { describe, expect, it } from 'vitest'
import type { Parameter } from './db'
import { formatTimestamp, formatValue } from './format'

function parameter(decimals: number, unit: string): Parameter {
  return { id: 1, shortname: 'x', description: 'x', unit, decimals }
}

describe('formatValue', () => {
  it('formats with declared decimals and unit', () => {
    expect(formatValue(12.34, parameter(1, 'mm'))).toBe('12.3 mm')
  })

  it('omits the unit when none is present', () => {
    expect(formatValue(5, parameter(0, ''))).toBe('5')
  })

  it('handles a null parameter', () => {
    expect(formatValue(3.14159, null)).toBe('3')
  })
})

describe('formatTimestamp', () => {
  it('formats an RFC3339 UTC timestamp', () => {
    expect(formatTimestamp('2026-08-16T00:00:00Z')).toBe('2026-08-16 00:00 UTC')
  })

  it('returns the input unchanged when unparsable', () => {
    expect(formatTimestamp('nonsense')).toBe('nonsense')
  })
})
