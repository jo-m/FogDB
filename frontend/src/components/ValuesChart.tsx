import {
  Bar,
  BarChart,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis
} from 'recharts'
import type { Parameter } from '../lib/db'
import { formatValue } from '../lib/format'

/** One bar in the chart: a location, its value, and its viridis color. */
export interface ChartDatum {
  label: string
  value: number
  color: string
}

interface ValuesChartProps {
  data: ChartDatum[]
  parameter: Parameter | null
}

interface TooltipProps {
  active?: boolean
  payload?: Array<{ payload: ChartDatum }>
}

function ChartTooltip({ active, payload, parameter }: TooltipProps & { parameter: Parameter | null }) {
  if (!active || !payload || payload.length === 0) return null
  const datum = payload[0].payload
  return (
    <div
      style={{
        background: 'white',
        border: '1px solid #ccc',
        borderRadius: 4,
        padding: '6px 10px',
        fontSize: 12
      }}
    >
      <div>{datum.label}</div>
      <div style={{ fontWeight: 600 }}>{formatValue(datum.value, parameter)}</div>
    </div>
  )
}

/**
 * Bar chart of every location's value at the selected timestamp. Bars are
 * colored with the same viridis scale used on the map.
 */
export default function ValuesChart({ data, parameter }: ValuesChartProps) {
  return (
    <div style={{ width: '100%', height: 300 }}>
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={data} margin={{ top: 8, right: 8, bottom: 4, left: 8 }} barCategoryGap={0}>
          <XAxis dataKey="label" tick={false} axisLine={false} tickLine={false} height={2} />
          <YAxis
            tick={{ fontSize: 11 }}
            width={44}
            label={
              parameter?.unit
                ? {
                    value: parameter.unit,
                    angle: -90,
                    position: 'insideLeft',
                    style: { fontSize: 11, textAnchor: 'middle' }
                  }
                : undefined
            }
          />
          <Tooltip cursor={false} content={<ChartTooltip parameter={parameter} />} />
          <Bar dataKey="value" isAnimationActive={false}>
            {data.map((d, i) => (
              <Cell key={i} fill={d.color} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  )
}
