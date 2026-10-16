/**
 * MeteoSwiss weather pictograms are published as SVGs under a fixed base URL,
 * one file per pictogram number. The forecast value of a pictogram parameter
 * is the pictogram number, which maps directly to a file name. We reference
 * the SVGs by URL rather than copying them, as they are copyrighted.
 */

/** Base URL of the MeteoSwiss weather symbol SVGs. */
const PICTOGRAM_BASE_URL =
  'https://www.meteoschweiz.admin.ch/static/resources/weather-symbols'

/** Parameter shortnames whose forecast value is a MeteoSwiss pictogram number. */
const PICTOGRAM_SHORTNAMES: ReadonlySet<string> = new Set(['jww003i0', 'jp2000d0'])

/**
 * Report whether a parameter stores a MeteoSwiss pictogram number.
 *
 * @param shortname The parameter shortname.
 * @returns True when the parameter's values are pictogram numbers.
 */
export function isPictogramParameter(shortname: string): boolean {
  return PICTOGRAM_SHORTNAMES.has(shortname)
}

/**
 * Build the URL of the MeteoSwiss pictogram SVG for a pictogram number.
 *
 * @param value The pictogram number.
 * @returns The absolute URL of the SVG.
 */
export function pictogramUrl(value: number): string {
  return `${PICTOGRAM_BASE_URL}/${Math.round(value)}.svg`
}
