import type { CSSProperties } from 'react'
import type { Parameter } from '../lib/db'
import { formatTimestamp } from '../lib/format'
import ColorLegend from './ColorLegend'
import ValuesChart, { type ChartDatum } from './ValuesChart'

interface SidePanelProps {
  parameters: Parameter[]
  selectedParameter: Parameter | null
  onSelectParameter: (parameter: Parameter) => void
  timestamps: string[]
  timeIndex: number
  onTimeIndexChange: (index: number) => void
  chartData: ChartDatum[]
  sortAsc: boolean
  onToggleSort: () => void
  min: number
  max: number
  valueCount: number
}

const sectionStyle: CSSProperties = {
  padding: '14px 16px',
  borderBottom: '1px solid #e2e5e8',
  background: 'white'
}

const headingStyle: CSSProperties = {
  margin: '0 0 10px',
  fontSize: 13,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: 0.4,
  color: '#666'
}

/**
 * Right-hand panel listing available parameters and, once one is selected,
 * the time slider, sort control, color legend and value chart.
 */
export default function SidePanel(props: SidePanelProps) {
  const {
    parameters,
    selectedParameter,
    onSelectParameter,
    timestamps,
    timeIndex,
    onTimeIndexChange,
    chartData,
    sortAsc,
    onToggleSort,
    min,
    max,
    valueCount
  } = props

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      <div style={{ ...sectionStyle, flex: '0 0 auto' }}>
        <h2 style={headingStyle}>Parameters</h2>
        <div style={{ maxHeight: 220, overflowY: 'auto' }}>
          {parameters.map((p) => {
            const active = selectedParameter?.id === p.id
            return (
              <button
                key={p.id}
                onClick={() => onSelectParameter(p)}
                style={{
                  display: 'block',
                  width: '100%',
                  textAlign: 'left',
                  padding: '8px 10px',
                  marginBottom: 4,
                  border: '1px solid transparent',
                  borderRadius: 6,
                  background: active ? '#e3f2fd' : '#f7f8f9',
                  borderColor: active ? '#2196f3' : '#e2e5e8',
                  cursor: 'pointer'
                }}
              >
                <div style={{ fontSize: 13, fontWeight: 600 }}>{p.shortname}</div>
                <div style={{ fontSize: 12, color: '#555' }}>{p.description}</div>
              </button>
            )
          })}
        </div>
      </div>

      {selectedParameter && timestamps.length > 0 && (
        <>
          <div style={{ ...sectionStyle, flex: '0 0 auto' }}>
            <h2 style={headingStyle}>Time</h2>
            <div style={{ fontSize: 13, fontWeight: 600, marginBottom: 8 }}>
              {formatTimestamp(timestamps[timeIndex])}
            </div>
            <input
              type="range"
              min={0}
              max={timestamps.length - 1}
              step={1}
              value={timeIndex}
              onChange={(e) => onTimeIndexChange(Number(e.target.value))}
              style={{ width: '100%' }}
            />
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                fontSize: 11,
                color: '#888',
                marginTop: 4
              }}
            >
              <span>{formatTimestamp(timestamps[0])}</span>
              <span>{formatTimestamp(timestamps[timestamps.length - 1])}</span>
            </div>
          </div>

          <div style={{ ...sectionStyle, flex: '0 0 auto' }}>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                marginBottom: 10
              }}
            >
              <span style={{ fontSize: 13 }}>
                {valueCount} locations
              </span>
              <button
                onClick={onToggleSort}
                style={{
                  padding: '6px 12px',
                  fontSize: 12,
                  border: '1px solid #c4c8cc',
                  borderRadius: 6,
                  background: 'white',
                  cursor: 'pointer'
                }}
              >
                Sort {sortAsc ? 'descending' : 'ascending'}
              </button>
            </div>
            <ColorLegend min={min} max={max} parameter={selectedParameter} />
          </div>

          <div style={{ flex: '1 1 auto', minHeight: 300, background: 'white', padding: '8px 16px 16px' }}>
            <ValuesChart data={chartData} parameter={selectedParameter} />
          </div>
        </>
      )}
    </div>
  )
}
