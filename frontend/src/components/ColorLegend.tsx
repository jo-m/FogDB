import type { Parameter } from '../lib/db'
import { formatValue } from '../lib/format'
import { viridisGradient } from '../lib/viridis'

interface ColorLegendProps {
  min: number
  max: number
  parameter: Parameter | null
}

/**
 * Horizontal viridis color bar with the value domain annotated at its ends
 * and midpoint.
 */
export default function ColorLegend({ min, max, parameter }: ColorLegendProps) {
  const mid = (min + max) / 2
  return (
    <div>
      <div
        style={{
          height: 12,
          borderRadius: 3,
          background: viridisGradient(64)
        }}
      />
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          marginTop: 4,
          fontSize: 11,
          color: '#444'
        }}
      >
        <span>{formatValue(min, parameter)}</span>
        <span>{formatValue(mid, parameter)}</span>
        <span>{formatValue(max, parameter)}</span>
      </div>
    </div>
  )
}
