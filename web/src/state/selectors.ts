import type { Alert, AlertType, Driver, Severity, Trip, Vehicle } from '../types'

// Condition — что показывает лампа машины. Приоритет: связь, тревога, статус.
export type Condition = 'offline' | 'alarm' | 'trip' | 'idle' | 'service'

export const CONDITION_LABEL: Record<Condition, string> = {
  offline: 'Нет связи',
  alarm: 'Тревога',
  trip: 'В рейсе',
  idle: 'Свободна',
  service: 'На обслуживании',
}

export const ALERT_LABEL: Record<AlertType, string> = {
  SPEED_EXCEEDED: 'Превышение скорости',
  LOW_FUEL: 'Мало топлива',
  DEVICE_OFFLINE: 'Нет связи с трекером',
}

const SEVERITY_RANK: Record<Severity, number> = { LOW: 0, MEDIUM: 1, HIGH: 2, CRITICAL: 3 }

export const severityRank = (s: Severity) => SEVERITY_RANK[s]

export function runningTrip(trips: Record<number, Trip>, vehicleId: number): Trip | undefined {
  for (const trip of Object.values(trips)) {
    if (trip.vehicle_id === vehicleId && trip.status === 'RUNNING') return trip
  }
  return undefined
}

// openAlertsOf — незакрытые алерты машины, самые критичные первыми
export function openAlertsOf(alerts: Record<number, Alert>, vehicleId: number): Alert[] {
  return Object.values(alerts)
    .filter((a) => a.vehicle_id === vehicleId && a.status !== 'RESOLVED')
    .sort((a, b) => SEVERITY_RANK[b.severity] - SEVERITY_RANK[a.severity] || b.id - a.id)
}

export function conditionOf(vehicle: Vehicle, open: Alert[]): Condition {
  if (open.some((a) => a.type === 'DEVICE_OFFLINE')) return 'offline'
  if (open.length > 0) return 'alarm'
  if (vehicle.status === 'ON_TRIP') return 'trip'
  if (vehicle.status === 'IN_SERVICE') return 'service'
  return 'idle'
}

export function conditionsById(
  vehicles: Record<number, Vehicle>,
  alerts: Record<number, Alert>,
): Map<number, Condition> {
  const open = new Map<number, Alert[]>()
  for (const a of Object.values(alerts)) {
    if (a.status === 'RESOLVED') continue
    open.set(a.vehicle_id, [...(open.get(a.vehicle_id) ?? []), a])
  }
  const result = new Map<number, Condition>()
  for (const v of Object.values(vehicles)) result.set(v.id, conditionOf(v, open.get(v.id) ?? []))
  return result
}

export interface Standing {
  driver: Driver
  km: number
  trips: number
  violations: number
  score: number
  // машина, на которой водитель сейчас в рейсе
  vehicleId: number | null
}

const PENALTY_KM = 50

// standings — рейтинг смены: пробег минус штраф за каждое превышение скорости.
// Превышение засчитывается водителю, чей рейс шёл на этой машине в момент алерта.
export function standings(
  drivers: Record<number, Driver>,
  trips: Record<number, Trip>,
  alerts: Record<number, Alert>,
  now: number,
): Standing[] {
  const byDriver = new Map<number, Standing>()
  for (const driver of Object.values(drivers)) {
    byDriver.set(driver.id, { driver, km: 0, trips: 0, violations: 0, score: 0, vehicleId: null })
  }

  const tripList = Object.values(trips).filter((t) => t.status !== 'CANCELLED')
  for (const trip of tripList) {
    const s = byDriver.get(trip.driver_id)
    if (!s) continue
    s.km += trip.distance_km
    s.trips += 1
    if (trip.status === 'RUNNING') s.vehicleId = trip.vehicle_id
  }

  for (const alert of Object.values(alerts)) {
    if (alert.type !== 'SPEED_EXCEEDED') continue
    const at = Date.parse(alert.created_at)
    const trip = tripList.find(
      (t) =>
        t.vehicle_id === alert.vehicle_id &&
        Date.parse(t.started_at) <= at &&
        at <= (t.ended_at ? Date.parse(t.ended_at) : now),
    )
    const s = trip && byDriver.get(trip.driver_id)
    if (s) s.violations += 1
  }

  return [...byDriver.values()]
    .map((s) => ({ ...s, score: Math.round(s.km - PENALTY_KM * s.violations) }))
    .sort((a, b) => b.score - a.score || a.driver.name.localeCompare(b.driver.name, 'ru'))
}
