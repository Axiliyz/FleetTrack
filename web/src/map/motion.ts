// Интерполяция движения, как в сетевых играх: точки приходят раз в секунду,
// а фура плавно доезжает от текущего положения до новой точки за тот же интервал.
// Состояние живёт вне React, чтобы кадр анимации не вызывал ререндер.
import { bearingDeg, haversineKm, lerpLngLat, type LngLat } from '../lib/geo'

const TRAIL_MS = 75_000
const TRAIL_MAX_POINTS = 200
const MIN_GLIDE_MS = 250
const MAX_GLIDE_MS = 1500
// при скачке дальше этого машину переносим сразу, без проезда сквозь карту
const TELEPORT_KM = 150

interface Track {
  from: LngLat
  to: LngLat
  t0: number
  dur: number
  bearing: number
  lastArrival: number
  trail: Array<{ pos: LngLat; t: number }>
}

export interface TrailSegment {
  id: number
  from: LngLat
  to: LngLat
  // 1 — свежий участок у машины, 0 — хвост, который вот-вот исчезнет
  strength: number
}

const tracks = new Map<number, Track>()

// часы анимации стоят на паузе вместе с симуляцией, чтобы следы не гасли
let elapsed = 0
let lastTick = performance.now()
let paused = false

function now(): number {
  const t = performance.now()
  if (!paused) elapsed += t - lastTick
  lastTick = t
  return elapsed
}

function positionAt(track: Track, t: number): LngLat {
  const k = Math.min(1, Math.max(0, (t - track.t0) / track.dur))
  return lerpLngLat(track.from, track.to, k)
}

export const motion = {
  now,

  setPaused(value: boolean) {
    now()
    paused = value
  },

  reset() {
    tracks.clear()
  },

  push(id: number, pos: LngLat) {
    const t = now()
    const track = tracks.get(id)
    if (!track) {
      tracks.set(id, { from: pos, to: pos, t0: t, dur: 1, bearing: 0, lastArrival: t, trail: [{ pos, t }] })
      return
    }

    const current = positionAt(track, t)
    const jump = haversineKm(current, pos)
    if (jump > 0.01) track.bearing = bearingDeg(current, pos)

    track.from = jump > TELEPORT_KM ? pos : current
    track.to = pos
    track.t0 = t
    track.dur = Math.min(MAX_GLIDE_MS, Math.max(MIN_GLIDE_MS, t - track.lastArrival))
    track.lastArrival = t

    if (jump > TELEPORT_KM) track.trail = []
    // в след кладём место, где машина реально была, а не цель — иначе хвост убежит вперёд
    track.trail.push({ pos: track.from, t })
    while (track.trail.length > TRAIL_MAX_POINTS || (track.trail.length > 0 && t - track.trail[0].t > TRAIL_MS)) {
      track.trail.shift()
    }
  },

  sample(id: number, t = now()): { pos: LngLat; bearing: number } | null {
    const track = tracks.get(id)
    return track ? { pos: positionAt(track, t), bearing: track.bearing } : null
  },

  trailSegments(t = now()): TrailSegment[] {
    const segments: TrailSegment[] = []
    for (const [id, track] of tracks) {
      const points = [...track.trail, { pos: positionAt(track, t), t }]
      for (let i = 1; i < points.length; i++) {
        const strength = 1 - (t - points[i].t) / TRAIL_MS
        if (strength <= 0) continue
        segments.push({ id, from: points[i - 1].pos, to: points[i].pos, strength })
      }
    }
    return segments
  },
}
