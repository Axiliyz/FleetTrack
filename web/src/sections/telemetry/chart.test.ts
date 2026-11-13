import { describe, expect, it } from 'vitest'
import type { TelemetryEvent } from '../../types'
import { GAP_MS, Y_STEP, buildPlot, linePath, nearest } from './chart'

const T0 = Date.parse('2026-10-20T10:00:00Z')

const point = (offsetMs: number, speed: number): TelemetryEvent => ({
  telemetry_id: offsetMs,
  vehicle_id: 1,
  device_id: 1,
  lat: 55,
  lon: 37,
  fuel: 50,
  received_at: new Date(T0 + offsetMs).toISOString(),
  trip_id: 1,
  distance_km: 0,
  speed_kmh: speed,
})

describe('buildPlot', () => {
  it('верх шкалы — с запасом над пиком и над порогом, кратно шагу', () => {
    expect(buildPlot([point(0, 40), point(1000, 72)], 400, undefined).yMax).toBe(80)
    // порог выше пика: пунктир не должен упираться в край
    expect(buildPlot([point(0, 40), point(1000, 50)], 400, 90).yMax).toBe(100)
    expect(buildPlot([point(0, 0), point(1000, 0)], 400, undefined).yMax % Y_STEP).toBe(0)
  })
})

describe('linePath', () => {
  it('рвёт линию там, где трекер молчал', () => {
    const points = [point(0, 50), point(1000, 60), point(1000 + GAP_MS + 1, 70), point(2000 + GAP_MS, 80)]
    const d = linePath(points, buildPlot(points, 400, undefined))
    expect(d.match(/M/g)).toHaveLength(2)
    expect(d.match(/L/g)).toHaveLength(2)
  })
})

describe('nearest', () => {
  const times = [0, 10, 20, 30]

  it('находит ближайшую по времени точку', () => {
    expect(nearest(times, 14)).toBe(1)
    expect(nearest(times, 16)).toBe(2)
  })

  it('за краями — крайние точки', () => {
    expect(nearest(times, -5)).toBe(0)
    expect(nearest(times, 99)).toBe(3)
  })
})
