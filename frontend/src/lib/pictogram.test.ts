import { describe, expect, it } from 'vitest'
import { isPictogramParameter, pictogramUrl } from './pictogram'

describe('isPictogramParameter', () => {
  it('recognises the pictogram shortnames', () => {
    expect(isPictogramParameter('jww003i0')).toBe(true)
    expect(isPictogramParameter('jp2000d0')).toBe(true)
  })

  it('rejects non-pictogram shortnames', () => {
    expect(isPictogramParameter('tre200h0')).toBe(false)
    expect(isPictogramParameter('rre150h0')).toBe(false)
  })
})

describe('pictogramUrl', () => {
  it('maps a pictogram number to its SVG URL', () => {
    expect(pictogramUrl(26)).toBe(
      'https://www.meteoschweiz.admin.ch/static/resources/weather-symbols/26.svg'
    )
  })

  it('rounds fractional values to the nearest pictogram number', () => {
    expect(pictogramUrl(2.4)).toBe(
      'https://www.meteoschweiz.admin.ch/static/resources/weather-symbols/2.svg'
    )
  })
})
