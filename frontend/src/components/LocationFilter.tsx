import { useMemo, useState } from 'react'
import type { Location } from '../lib/db'
import { searchLocations } from '../lib/search'

interface LocationFilterProps {
  locations: Location[]
  filteredCount: number
  viewportFilter: boolean
  onToggleViewport: () => void
  manualFilter: boolean
  onToggleManual: () => void
  manualPointIds: number[]
  onAddPointId: (pointId: number) => void
  onRemovePointId: (pointId: number) => void
  onClearManual: () => void
}

const headingStyle = {
  margin: '0 0 10px',
  fontSize: 13,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: 0.4,
  color: '#666'
} as const

const checkboxRowStyle = {
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  fontSize: 13,
  marginBottom: 8
} as const

const chipStyle = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: 6,
  padding: '3px 8px',
  margin: '0 6px 6px 0',
  border: '1px solid #c4c8cc',
  borderRadius: 999,
  background: '#f0f2f4',
  fontSize: 12
} as const

/**
 * Controls for narrowing the set of visible locations: a viewport filter, and
 * a manually maintained list that can be filled via fuzzy search or map click.
 */
export default function LocationFilter(props: LocationFilterProps) {
  const {
    locations,
    filteredCount,
    viewportFilter,
    onToggleViewport,
    manualFilter,
    onToggleManual,
    manualPointIds,
    onAddPointId,
    onRemovePointId,
    onClearManual
  } = props

  const [query, setQuery] = useState('')

  const locationByPointId = useMemo(() => {
    const map = new Map<number, Location>()
    for (const location of locations) {
      if (!map.has(location.pointId)) map.set(location.pointId, location)
    }
    return map
  }, [locations])

  const manualSet = useMemo(() => new Set(manualPointIds), [manualPointIds])

  const results = useMemo(() => searchLocations(locations, query, 8), [locations, query])

  const manualLocations = useMemo(
    () =>
      manualPointIds
        .map((id) => locationByPointId.get(id))
        .filter((l): l is Location => l !== undefined),
    [manualPointIds, locationByPointId]
  )

  return (
    <div>
      <h2 style={headingStyle}>Locations</h2>

      <div style={{ fontSize: 13, marginBottom: 10 }}>
        {filteredCount} of {locations.length} locations shown
      </div>

      <label style={checkboxRowStyle}>
        <input
          type="checkbox"
          checked={viewportFilter}
          onChange={onToggleViewport}
          style={{ margin: 0 }}
        />
        Filter for current map viewport
      </label>

      <label style={checkboxRowStyle}>
        <input
          type="checkbox"
          checked={manualFilter}
          onChange={onToggleManual}
          disabled={manualPointIds.length === 0}
          style={{ margin: 0 }}
        />
        Limit to selected locations ({manualPointIds.length})
      </label>

      <input
        type="text"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Search to add a location"
        style={{
          width: '100%',
          padding: '6px 10px',
          fontSize: 13,
          border: '1px solid #c4c8cc',
          borderRadius: 6,
          marginBottom: 8
        }}
      />

      {query.trim().length > 0 && (
        <div
          style={{
            border: '1px solid #e2e5e8',
            borderRadius: 6,
            marginBottom: 8,
            overflow: 'hidden'
          }}
        >
          {results.length === 0 && (
            <div style={{ padding: '8px 10px', fontSize: 12, color: '#888' }}>No matches</div>
          )}
          {results.map((location) => {
            const added = manualSet.has(location.pointId)
            return (
              <button
                key={location.pointId}
                onClick={() => {
                  onAddPointId(location.pointId)
                  setQuery('')
                }}
                disabled={added}
                style={{
                  display: 'block',
                  width: '100%',
                  textAlign: 'left',
                  padding: '7px 10px',
                  border: 'none',
                  borderBottom: '1px solid #eef0f2',
                  background: 'white',
                  fontSize: 13,
                  cursor: added ? 'default' : 'pointer',
                  opacity: added ? 0.5 : 1
                }}
              >
                {location.name}
                {location.abbr ? ` (${location.abbr})` : ''}
                {added ? ' - added' : ''}
              </button>
            )
          })}
        </div>
      )}

      {manualLocations.length > 0 && (
        <div>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              marginBottom: 6
            }}
          >
            <span style={{ fontSize: 12, fontWeight: 600, color: '#555' }}>Selected</span>
            <button
              onClick={onClearManual}
              style={{
                padding: '2px 8px',
                fontSize: 12,
                border: '1px solid #c4c8cc',
                borderRadius: 6,
                background: 'white',
                cursor: 'pointer'
              }}
            >
              Clear
            </button>
          </div>
          <div>
            {manualLocations.map((location) => (
              <span key={location.pointId} style={chipStyle}>
                {location.name}
                <button
                  onClick={() => onRemovePointId(location.pointId)}
                  aria-label={`Remove ${location.name}`}
                  style={{
                    border: 'none',
                    background: 'transparent',
                    cursor: 'pointer',
                    padding: 0,
                    fontSize: 13,
                    lineHeight: 1,
                    color: '#666'
                  }}
                >
                  x
                </button>
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
