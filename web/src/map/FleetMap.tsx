import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { Map as MapLibreMap, Marker } from 'maplibre-gl'
import { MapboxOverlay } from '@deck.gl/mapbox'
import { useFleet } from '../state/fleet'
import { conditionsById, type Condition } from '../state/selectors'
import { STYLE_URL } from './basemap'
import { buildLayers, NEAR_ZOOM } from './layers'
import { motion } from './motion'
import { MOSCOW, NETWORK_BOUNDS } from './network'
import { TruckMarker } from './TruckMarker'

const reducedMotion = () => window.matchMedia('(prefers-reduced-motion: reduce)').matches

interface MarkerHandle {
  marker: Marker
  el: HTMLDivElement
}

export function FleetMap() {
  const containerRef = useRef<HTMLDivElement>(null)
  const mapRef = useRef<MapLibreMap | null>(null)
  const markersRef = useRef(new Map<number, MarkerHandle>())
  const [portals, setPortals] = useState<Array<[number, HTMLDivElement]>>([])
  const vehicles = useFleet((s) => s.vehicles)
  const camera = useFleet((s) => s.camera)
  const followId = useFleet((s) => s.followId)

  // карта, слой deck.gl и цикл анимации
  useEffect(() => {
    const container = containerRef.current
    if (!container) return
    const narrow = container.clientWidth < 700

    const map = new MapLibreMap({
      container,
      style: STYLE_URL,
      center: MOSCOW.pos,
      zoom: 5,
      attributionControl: { compact: true },
      dragRotate: false,
      pitchWithRotate: false,
      renderWorldCopies: false,
    })
    map.touchZoomRotate.disableRotation()
    // снизу оставляем место под журнал, чтобы он не закрывал южные трассы
    map.fitBounds(NETWORK_BOUNDS, {
      padding: narrow ? 24 : { top: 48, bottom: 200, left: 48, right: 48 },
      animate: false,
    })
    mapRef.current = map

    const overlay = new MapboxOverlay({ interleaved: false, layers: [] })
    map.addControl(overlay)

    // если диспетчер сам двигает карту, слежение за машиной отключается
    map.on('dragstart', () => useFleet.getState().setFollow(null))
    const syncZoom = () => {
      container.dataset.zoom = map.getZoom() >= NEAR_ZOOM ? 'near' : 'far'
    }
    map.on('zoom', syncZoom)
    syncZoom()

    let fontsReady = false
    document.fonts.load('500 13px "IBM Plex Sans Condensed"').finally(() => {
      fontsReady = true
    })

    // состояния ламп пересчитываем только когда меняются машины или алерты
    let cache: { vehicles: unknown; alerts: unknown; map: Map<number, Condition> } | null = null
    const conditionOf = (id: number): Condition => {
      const { vehicles, alerts } = useFleet.getState()
      if (!cache || cache.vehicles !== vehicles || cache.alerts !== alerts) {
        cache = { vehicles, alerts, map: conditionsById(vehicles, alerts) }
      }
      return cache.map.get(id) ?? 'idle'
    }

    let raf = 0
    const frame = () => {
      const t = motion.now()
      for (const [id, { marker, el }] of markersRef.current) {
        const s = motion.sample(id, t)
        if (!s) continue
        marker.setLngLat(s.pos)
        el.style.setProperty('--heading', `${s.bearing}deg`)
      }

      const { followId } = useFleet.getState()
      // пока идёт перелёт камеры, не перебиваем его
      if (followId !== null && !map.isMoving()) {
        const s = motion.sample(followId, t)
        if (s) map.setCenter(s.pos)
      }

      overlay.setProps({ layers: buildLayers({ zoom: map.getZoom(), fontsReady, conditionOf }) })
      raf = requestAnimationFrame(frame)
    }
    raf = requestAnimationFrame(frame)

    const markers = markersRef.current
    return () => {
      cancelAnimationFrame(raf)
      markers.clear()
      map.remove()
      mapRef.current = null
      setPortals([])
    }
  }, [])

  // маркеры появляются и исчезают вместе с машинами в сторе
  useEffect(() => {
    const map = mapRef.current
    if (!map) return
    const markers = markersRef.current
    let changed = false

    for (const vehicle of Object.values(vehicles)) {
      if (markers.has(vehicle.id)) continue
      const el = document.createElement('div')
      el.className = 'unit-anchor'
      const pos = motion.sample(vehicle.id)?.pos ?? MOSCOW.pos
      const marker = new Marker({ element: el }).setLngLat(pos).addTo(map)
      markers.set(vehicle.id, { marker, el })
      changed = true
    }
    for (const [id, handle] of markers) {
      if (vehicles[id]) continue
      handle.marker.remove()
      markers.delete(id)
      changed = true
    }
    if (changed) setPortals([...markers].map(([id, { el }]) => [id, el]))
  }, [vehicles])

  // перелёт к выбранной машине
  useEffect(() => {
    const map = mapRef.current
    if (!map || !camera) return
    const s = motion.sample(camera.vehicleId)
    if (!s) return
    const target = { center: s.pos, zoom: Math.max(map.getZoom(), 7.5) }
    if (reducedMotion()) map.jumpTo(target)
    else map.flyTo({ ...target, speed: 1.6 })
  }, [camera])

  // при включении слежения плавно подводим камеру, дальше её ведёт цикл анимации
  useEffect(() => {
    const map = mapRef.current
    if (!map || followId === null) return
    const s = motion.sample(followId)
    if (!s) return
    const target = { center: s.pos, zoom: Math.max(map.getZoom(), 8) }
    if (reducedMotion()) map.jumpTo(target)
    else map.easeTo({ ...target, duration: 600 })
  }, [followId])

  return (
    <div ref={containerRef} className="map" data-zoom="far">
      {portals.map(([id, el]) => createPortal(<TruckMarker vehicleId={id} />, el, id))}
    </div>
  )
}
