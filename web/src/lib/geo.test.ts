import { describe, expect, it } from 'vitest'
import { bearingDeg, buildRoute, haversineKm, pointAlong, type LngLat } from './geo'

const MOSCOW: LngLat = [37.6173, 55.7558]
const SPB: LngLat = [30.3158, 59.9391]

describe('haversineKm', () => {
  it('считает расстояние Москва — Петербург по прямой', () => {
    expect(haversineKm(MOSCOW, SPB)).toBeGreaterThan(630)
    expect(haversineKm(MOSCOW, SPB)).toBeLessThan(640)
  })

  it('возвращает ноль для одной точки', () => {
    expect(haversineKm(MOSCOW, MOSCOW)).toBe(0)
  })
})

describe('bearingDeg', () => {
  it.each<[string, LngLat, number]>([
    ['север', [37.6173, 56.7558], 0],
    ['восток', [38.6173, 55.7558], 90],
    ['юг', [37.6173, 54.7558], 180],
    ['запад', [36.6173, 55.7558], 270],
  ])('%s', (_, to, expected) => {
    expect(bearingDeg(MOSCOW, to)).toBeCloseTo(expected, -0.5)
  })
})

describe('pointAlong', () => {
  const route = buildRoute([
    [0, 0],
    [0, 1],
    [1, 1],
  ])

  it('ограничивает пробег началом и концом маршрута', () => {
    expect(pointAlong(route, -5).pos).toEqual([0, 0])
    expect(pointAlong(route, route.lengthKm + 5).pos).toEqual([1, 1])
  })

  it('находит точку на втором отрезке и его курс', () => {
    const first = route.cumKm[1]
    const { pos, bearing } = pointAlong(route, first + (route.lengthKm - first) / 2)
    expect(pos[0]).toBeCloseTo(0.5, 5)
    expect(pos[1]).toBeCloseTo(1, 5)
    expect(bearing).toBeCloseTo(90, 0)
  })

  it('отказывается строить маршрут из одной точки', () => {
    expect(() => buildRoute([[0, 0]])).toThrow()
  })
})
