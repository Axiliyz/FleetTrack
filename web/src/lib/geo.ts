// Геометрия на сфере: те же гаверсинусы, что и в MotionService на бэке.

// LngLat — [долгота, широта], порядок как в MapLibre и deck.gl
export type LngLat = [number, number]

const EARTH_RADIUS_KM = 6371

const toRad = (deg: number) => (deg * Math.PI) / 180
const toDeg = (rad: number) => (rad * 180) / Math.PI

// haversineKm возвращает расстояние между точками по дуге большого круга
export function haversineKm([lon1, lat1]: LngLat, [lon2, lat2]: LngLat): number {
  const dLat = toRad(lat2 - lat1)
  const dLon = toRad(lon2 - lon1)
  const h =
    Math.sin(dLat / 2) ** 2 + Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLon / 2) ** 2
  return 2 * EARTH_RADIUS_KM * Math.asin(Math.sqrt(h))
}

// bearingDeg возвращает курс от a к b в градусах: 0 — север, 90 — восток
export function bearingDeg([lon1, lat1]: LngLat, [lon2, lat2]: LngLat): number {
  const phi1 = toRad(lat1)
  const phi2 = toRad(lat2)
  const dLambda = toRad(lon2 - lon1)
  const y = Math.sin(dLambda) * Math.cos(phi2)
  const x = Math.cos(phi1) * Math.sin(phi2) - Math.sin(phi1) * Math.cos(phi2) * Math.cos(dLambda)
  return (toDeg(Math.atan2(y, x)) + 360) % 360
}

export function lerpLngLat(a: LngLat, b: LngLat, t: number): LngLat {
  return [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t]
}

// Route — ломаная с накопленными расстояниями, чтобы быстро находить точку по пробегу
export interface Route {
  points: LngLat[]
  cumKm: number[]
  lengthKm: number
}

export function buildRoute(points: LngLat[]): Route {
  if (points.length < 2) throw new Error('маршрут должен состоять минимум из двух точек')
  const cumKm = [0]
  for (let i = 1; i < points.length; i++) {
    cumKm.push(cumKm[i - 1] + haversineKm(points[i - 1], points[i]))
  }
  return { points, cumKm, lengthKm: cumKm[cumKm.length - 1] }
}

// pointAlong возвращает точку и курс на расстоянии km от начала маршрута
export function pointAlong(route: Route, km: number): { pos: LngLat; bearing: number } {
  const d = Math.min(Math.max(km, 0), route.lengthKm)
  let i = 1
  while (i < route.cumKm.length - 1 && route.cumKm[i] < d) i++
  const a = route.points[i - 1]
  const b = route.points[i]
  const segment = route.cumKm[i] - route.cumKm[i - 1]
  const t = segment > 0 ? (d - route.cumKm[i - 1]) / segment : 0
  return { pos: lerpLngLat(a, b, t), bearing: bearingDeg(a, b) }
}
