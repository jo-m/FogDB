import { useCallback, useMemo, useRef, useState } from 'react'
import type { MouseEvent as ReactMouseEvent } from 'react'
import FileUpload from './components/FileUpload'
import MapView, { type MapPoint } from './components/MapView'
import SidePanel from './components/SidePanel'
import type { ChartDatum } from './components/ValuesChart'
import {
  getTimestamps,
  getValues,
  loadDatabase,
  type LoadedDb,
  type LocationValue,
  type Parameter
} from './lib/db'
import { makeViridisScale } from './lib/viridis'

export default function App() {
  const [showUpload, setShowUpload] = useState(true)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [loaded, setLoaded] = useState<LoadedDb | null>(null)

  const [selectedParameter, setSelectedParameter] = useState<Parameter | null>(null)
  const [timestamps, setTimestamps] = useState<string[]>([])
  const [timeIndex, setTimeIndex] = useState(0)
  const [values, setValues] = useState<LocationValue[]>([])
  const [sortAsc, setSortAsc] = useState(true)

  const handleFile = useCallback(async (file: File) => {
    setLoading(true)
    setError(null)
    try {
      const next = await loadDatabase(file)
      setLoaded((prev) => {
        if (prev) prev.db.close()
        return next
      })
      setSelectedParameter(null)
      setTimestamps([])
      setTimeIndex(0)
      setValues([])
      setSortAsc(true)
      setShowUpload(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setLoading(false)
    }
  }, [])

  const handleSelectParameter = useCallback(
    (parameter: Parameter) => {
      if (!loaded) return
      if (selectedParameter?.id === parameter.id) {
        setSelectedParameter(null)
        setTimestamps([])
        setTimeIndex(0)
        setValues([])
        return
      }
      const ts = getTimestamps(loaded.db, parameter.id)
      setSelectedParameter(parameter)
      setTimestamps(ts)
      setTimeIndex(ts.length - 1)
      setValues(getValues(loaded.db, parameter.id, ts[ts.length - 1], loaded.locationById))
    },
    [loaded, selectedParameter]
  )

  const handleTimeIndexChange = useCallback(
    (index: number) => {
      if (!loaded || !selectedParameter || !timestamps[index]) return
      setTimeIndex(index)
      setValues(getValues(loaded.db, selectedParameter.id, timestamps[index], loaded.locationById))
    },
    [loaded, selectedParameter, timestamps]
  )

  const { min, max } = useMemo(() => {
    if (values.length === 0) return { min: 0, max: 0 }
    let lo = Infinity
    let hi = -Infinity
    for (const v of values) {
      if (v.value < lo) lo = v.value
      if (v.value > hi) hi = v.value
    }
    return { min: lo, max: hi }
  }, [values])

  const scale = useMemo(() => makeViridisScale(min, max), [min, max])

  const points = useMemo<MapPoint[]>(() => {
    if (!loaded) return []
    if (!selectedParameter) {
      return loaded.locations.map((l) => ({ lat: l.lat, lon: l.lon, color: null }))
    }
    return values.map((v) => ({ lat: v.lat, lon: v.lon, color: scale(v.value) }))
  }, [loaded, selectedParameter, values, scale])

  const chartData = useMemo<ChartDatum[]>(() => {
    const sorted = [...values].sort((a, b) => (sortAsc ? a.value - b.value : b.value - a.value))
    return sorted.map((v) => ({
      label: v.name || v.abbr || String(v.locationId),
      value: v.value,
      color: scale(v.value)
    }))
  }, [values, sortAsc, scale])

  // Draggable splitter state.
  const splitRef = useRef<HTMLDivElement>(null)
  const [leftFrac, setLeftFrac] = useState(0.55)

  const startDrag = useCallback((event: ReactMouseEvent) => {
    event.preventDefault()
    const container = splitRef.current
    if (!container) return
    document.body.style.userSelect = 'none'

    const onMove = (ev: MouseEvent) => {
      const rect = container.getBoundingClientRect()
      const frac = (ev.clientX - rect.left) / rect.width
      setLeftFrac(Math.min(0.8, Math.max(0.2, frac)))
    }
    const onUp = () => {
      document.body.style.userSelect = ''
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
  }, [])

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {loaded && (
        <header
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '8px 16px',
            background: '#1c2733',
            color: 'white',
            flex: '0 0 auto'
          }}
        >
          <span style={{ fontWeight: 600 }}>FogDB Viewer</span>
          <button
            onClick={() => setShowUpload(true)}
            style={{
              padding: '6px 12px',
              fontSize: 12,
              border: '1px solid #4a5a6a',
              borderRadius: 6,
              background: 'transparent',
              color: 'white',
              cursor: 'pointer'
            }}
          >
            Load different database
          </button>
        </header>
      )}

      {loaded && (
        <div ref={splitRef} style={{ flex: 1, display: 'flex', minHeight: 0 }}>
          <div style={{ flex: leftFrac, minWidth: 0, position: 'relative', height: '100%' }}>
            <MapView points={points} />
          </div>
          <div
            onMouseDown={startDrag}
            style={{
              width: 8,
              cursor: 'col-resize',
              background: '#d6dadd',
              flex: '0 0 auto',
              borderLeft: '1px solid #c4c8cc',
              borderRight: '1px solid #c4c8cc'
            }}
          />
          <div style={{ flex: 1 - leftFrac, minWidth: 0 }}>
            <SidePanel
              parameters={loaded.parameters}
              selectedParameter={selectedParameter}
              onSelectParameter={handleSelectParameter}
              timestamps={timestamps}
              timeIndex={timeIndex}
              onTimeIndexChange={handleTimeIndexChange}
              chartData={chartData}
              sortAsc={sortAsc}
              onToggleSort={() => setSortAsc((v) => !v)}
              min={min}
              max={max}
              valueCount={values.length}
            />
          </div>
        </div>
      )}

      {showUpload && (
        <FileUpload
          onFile={handleFile}
          loading={loading}
          error={error}
          onCancel={loaded ? () => setShowUpload(false) : undefined}
        />
      )}
    </div>
  )
}
