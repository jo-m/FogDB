import { describe, expect, it } from 'vitest'
import { makeViridisScale, viridis, viridisGradient } from './viridis'

describe('viridis', () => {
  it('maps the domain endpoints to the first and last colormap entries', () => {
    expect(viridis(0)).toBe('rgb(68, 1, 84)')
    expect(viridis(1)).toBe('rgb(253, 231, 37)')
  })

  it('clamps out-of-range inputs', () => {
    expect(viridis(-1)).toBe(viridis(0))
    expect(viridis(2)).toBe(viridis(1))
  })

  it('always returns an rgb() string', () => {
    for (let i = 0; i <= 10; i++) {
      expect(viridis(i / 10)).toMatch(/^rgb\(\d+, \d+, \d+\)$/)
    }
  })
})

describe('makeViridisScale', () => {
  it('maps min and max to the colormap endpoints', () => {
    const scale = makeViridisScale(0, 10)
    expect(scale(0)).toBe(viridis(0))
    expect(scale(10)).toBe(viridis(1))
  })

  it('collapses a zero-width domain to the midpoint color', () => {
    const scale = makeViridisScale(5, 5)
    expect(scale(5)).toBe(viridis(0.5))
    expect(scale(3)).toBe(viridis(0.5))
  })
})

describe('viridisGradient', () => {
  it('produces a linear-gradient with one stop per sample', () => {
    const gradient = viridisGradient(4)
    expect(gradient.startsWith('linear-gradient(90deg,')).toBe(true)
    expect(gradient.split('rgb(').length - 1).toBe(5)
  })
})
