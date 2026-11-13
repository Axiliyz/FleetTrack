import { useEffect, useRef, useState } from 'react'
import { LngLatBounds, Map as MapLibreMap, type GeoJSONSource } from 'maplibre-gl'
import type { TelemetryEvent } from '../types'
import { STYLE_URL } from './basemap'
import { MOSCOW } from './network'

// цвета — те же, что у ламп в styles.css: линия трека «в рейсе», старт «свободна»
const LINE = '#f2a33a'
const START = '#6bd58f'
const FINISH = '#d6decf'
const BOARD = '#0e1613'

const EMPTY: GeoJSON.FeatureCollection = { type: 'FeatureCollection', features: [] }

function trackGeoJSON(points: TelemetryEvent[]): GeoJSON.FeatureCollection {
  if (points.length === 0) return EMPTY
  const coords = points.map((p) => [p.lon, p.lat])
  const ends: Array<[string, number[]]> = [
    ['start', coords[0]],
    ['finish', coords[coords.length - 1]],
  ]
  return {
    type: 'FeatureCollection',
    features: [
      { type: 'Feature', properties: {}, geometry: { type: 'LineString', coordinates: coords } },
      ...ends.map(([kind, pos]) => ({
        type: 'Feature' as const,
        properties: { kind },
        geometry: { type: 'Point' as const, coordinates: pos },
      })),
    ],
  }
}

// TrackMap — трек на карте. fitKey меняется, когда сменили машину, рейс или период:
// только тогда карта заново вписывает трек, иначе автообновление сбивало бы то, что человек приблизил
export function TrackMap({ points, fitKey }: { points: TelemetryEvent[]; fitKey: string }) {
  const containerRef = useRef<HTMLDivElement>(null)
  const mapRef = useRef<MapLibreMap | null>(null)
  const fittedRef = useRef<string | null>(null)
  const [ready, setReady] = useState(false)

  useEffect(() => {
    const container = containerRef.current
    if (!container) return
    const map = new MapLibreMap({
      container,
      style: STYLE_URL,
      center: MOSCOW.pos,
      zoom: 5,
      attributionControl: { compact: true },
      dragRotate: false,
      pitchWithRotate: false,
      // карта внутри прокручиваемой страницы: колесо листает страницу, зум — с Ctrl или двумя пальцами
      cooperativeGestures: true,
    })
    map.touchZoomRotate.disableRotation()
    map.on('load', () => {
      map.addSource('track', { type: 'geojson', data: EMPTY })
      map.addLayer({
        id: 'track-line',
        type: 'line',
        source: 'track',
        filter: ['==', ['geometry-type'], 'LineString'],
        layout: { 'line-join': 'round', 'line-cap': 'round' },
        paint: { 'line-color': LINE, 'line-width': 3 },
      })
      map.addLayer({
        id: 'track-ends',
        type: 'circle',
        source: 'track',
        filter: ['==', ['geometry-type'], 'Point'],
        paint: {
          'circle-radius': 6,
          'circle-color': ['match', ['get', 'kind'], 'start', START, FINISH],
          'circle-stroke-color': BOARD,
          'circle-stroke-width': 2,
        },
      })
      setReady(true)
    })
    mapRef.current = map
    return () => {
      map.remove()
      mapRef.current = null
      setReady(false)
    }
  }, [])

  useEffect(() => {
    const map = mapRef.current
    if (!map || !ready) return
    map.getSource<GeoJSONSource>('track')?.setData(trackGeoJSON(points))
    if (points.length === 0 || fittedRef.current === fitKey) return
    fittedRef.current = fitKey
    const bounds = new LngLatBounds()
    for (const p of points) bounds.extend([p.lon, p.lat])
    map.fitBounds(bounds, { padding: 40, maxZoom: 13, animate: false })
  }, [points, fitKey, ready])

  return <div ref={containerRef} className="map track-map" role="img" aria-label="Трек на карте" />
}
