import 'ol/ol.css'
import 'react-openlayers/dist/index.css'

import { useCallback, useEffect, useRef, useState } from 'react'
import proj4 from 'proj4'
import { XYZ, Vector as VectorSource } from 'ol/source'
import { register } from 'ol/proj/proj4'
import { transform } from 'ol/proj'
import { defaults as defaultControls, ScaleLine } from 'ol/control'
import { defaults as defaultInteractions } from 'ol/interaction'
import { Map, View, TileLayer, VectorLayer } from 'react-openlayers'
import { Feature } from 'ol'
import { Point as OLPoint } from 'ol/geom'
import { Style, Circle, Fill, Stroke } from 'ol/style'
import type { Map as OLMap } from 'ol'
import type MapBrowserEvent from 'ol/MapBrowserEvent'

import { LV95, registerProj4 } from '@swissgeo/coordinates'
import { getLV95TileGrid, getLV95ViewConfig } from '@swissgeo/coordinates/ol'
import type { WgsBbox } from '../lib/filter'

// Register the Swiss projections on proj4 and OpenLayers.
registerProj4(proj4)
register(proj4)

const mapSource = new XYZ({
  url: 'https://wmts.geo.admin.ch/1.0.0/ch.swisstopo.pixelkarte-farbe/default/current/2056/{z}/{x}/{y}.jpeg',
  projection: LV95.epsg,
  tileGrid: getLV95TileGrid()
})

const pointsSource = new VectorSource()

/** Neutral marker color used when no parameter is selected. */
const NEUTRAL_COLOR = 'rgba(40, 80, 120, 0.85)'

/** A single marker: a WGS84 position, an optional viridis color, and identity. */
export interface MapPoint {
  lat: number
  lon: number
  color: string | null
  pointId: number
  inList: boolean
}

interface MapViewProps {
  points: MapPoint[]
  onViewportChange: (bbox: WgsBbox | null) => void
  onPointClick: (pointId: number) => void
}

/**
 * Compute the current map viewport as a WGS84 bounding box.
 *
 * @param map The OpenLayers map.
 * @returns The visible extent in degrees, or null when the map has no size.
 */
function toWgsBbox(map: OLMap): WgsBbox | null {
  const size = map.getSize()
  if (!size || size[0] === 0 || size[1] === 0) return null
  const extent = map.getView().calculateExtent(size)
  const [minLon, minLat] = transform([extent[0], extent[1]], LV95.epsg, 'EPSG:4326')
  const [maxLon, maxLat] = transform([extent[2], extent[3]], LV95.epsg, 'EPSG:4326')
  return { minLat, minLon, maxLat, maxLon }
}

/**
 * SwissTopo base map overlaid with one marker per forecast location. Reports
 * viewport changes so the parent can filter by extent, and forwards marker
 * clicks so locations can be added to or removed from a manual list.
 */
export default function MapView({ points, onViewportChange, onPointClick }: MapViewProps) {
  const mapRef = useRef<OLMap | null>(null)
  const [mapReady, setMapReady] = useState(false)
  const hasFitted = useRef(false)

  // Rebuild the marker features whenever the points change.
  useEffect(() => {
    pointsSource.clear()
    for (const point of points) {
      const coords = transform([point.lon, point.lat], 'EPSG:4326', LV95.epsg)
      const feature = new Feature({ geometry: new OLPoint(coords) })
      feature.set('pointId', point.pointId)
      feature.setStyle(
        new Style({
          image: new Circle({
            radius: point.inList ? 7 : 5,
            fill: new Fill({ color: point.color ?? NEUTRAL_COLOR }),
            stroke: new Stroke({
              color: point.inList ? 'rgba(0, 0, 0, 0.9)' : 'rgba(255, 255, 255, 0.9)',
              width: point.inList ? 2 : 1
            })
          })
        })
      )
      pointsSource.addFeature(feature)
    }
  }, [points])

  const reportViewport = useCallback(() => {
    const map = mapRef.current
    if (map) onViewportChange(toWgsBbox(map))
  }, [onViewportChange])

  // Report viewport changes and forward marker clicks once the map is ready.
  useEffect(() => {
    const map = mapRef.current
    if (!map) return
    const onClick = (event: MapBrowserEvent) => {
      const feature = map.forEachFeatureAtPixel(
        event.pixel,
        (f) => f,
        { hitTolerance: 5 }
      ) as Feature | undefined
      const pointId = feature?.get('pointId')
      if (typeof pointId === 'number') onPointClick(pointId)
    }
    map.on('moveend', reportViewport)
    map.on('singleclick', onClick)
    return () => {
      map.un('moveend', reportViewport)
      map.un('singleclick', onClick)
    }
  }, [mapReady, reportViewport, onPointClick])

  // Fit the view to the marker extent once, after the first data appears.
  useEffect(() => {
    if (!mapReady || !mapRef.current || hasFitted.current || points.length === 0) return
    const extent = pointsSource.getExtent()
    if (extent && Number.isFinite(extent[0]) && Number.isFinite(extent[2])) {
      mapRef.current.getView().fit(extent, { padding: [24, 24, 24, 24] })
      hasFitted.current = true
      reportViewport()
    }
  }, [mapReady, points, reportViewport])

  return (
    <Map
      ref={(map) => {
        mapRef.current = map ?? null
        if (map && !mapReady) setMapReady(true)
      }}
      controls={defaultControls().extend([new ScaleLine({ units: 'metric' })])}
      interactions={defaultInteractions({ pinchRotate: false, altShiftDragRotate: false })}
      style={{ width: '100%', height: '100%' }}
    >
      <TileLayer source={mapSource} />
      <VectorLayer source={pointsSource} />
      <View {...getLV95ViewConfig()} enableRotation={false} constrainRotation={false} />
    </Map>
  )
}
