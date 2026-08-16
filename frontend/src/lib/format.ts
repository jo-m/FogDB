import type { Parameter } from './db'

/**
 * Format a forecast value using the parameter's declared decimals and unit.
 *
 * @param value The raw numeric value.
 * @param parameter The parameter metadata, or null when unknown.
 * @returns A display string such as `12.3 mm`.
 */
export function formatValue(value: number, parameter: Parameter | null): string {
  const decimals = parameter?.decimals ?? 0
  const number = value.toFixed(decimals)
  return parameter?.unit ? `${number} ${parameter.unit}` : number
}

/**
 * Format an RFC3339 UTC timestamp for display.
 *
 * @param timestamp The RFC3339 UTC timestamp string.
 * @returns A display string such as `2026-08-16 00:00 UTC`.
 */
export function formatTimestamp(timestamp: string): string {
  const date = new Date(timestamp)
  if (Number.isNaN(date.getTime())) return timestamp
  const y = date.getUTCFullYear()
  const m = String(date.getUTCMonth() + 1).padStart(2, '0')
  const d = String(date.getUTCDate()).padStart(2, '0')
  const hh = String(date.getUTCHours()).padStart(2, '0')
  const mm = String(date.getUTCMinutes()).padStart(2, '0')
  return `${y}-${m}-${d} ${hh}:${mm} UTC`
}
