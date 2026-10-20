// Симулятор парка: фуры ездят по федеральным трассам, шлют телеметрию,
// а правила алертов срабатывают так же, как в AlertService и OfflineWorker на бэке.
import type { FleetSource, SimControls, SubscribeHandlers } from '../FleetSource'
import type {
  Alert,
  AlertRule,
  AlertType,
  Driver,
  FleetEvent,
  FleetSnapshot,
  TelemetryEvent,
  Trip,
  Vehicle,
  VehicleStatus,
} from '../../types'
import { buildRoute, haversineKm, pointAlong, type LngLat, type Route } from '../../lib/geo'
import { createRng, type Rng } from '../../lib/random'
import { CORRIDORS, corridorPath } from '../../map/network'
import { CURRENT_USER_ID, ORG, makeDrivers, makeRules, makeVehicles } from './fixtures'

const MIN = 60_000
const HOUR = 60 * MIN

// шаг симуляции и период отправки точки трекером — в реальном времени
const TICK_MS = 250
const EMIT_EVERY_MS = 1000

const VEHICLE_COUNT = 14
// первые 11 в рейсе, затем две на стоянке и последняя на ТО
const DRIVING_COUNT = 11

// ~30 л на 100 км при баке 600 л
const FUEL_PCT_PER_KM = 0.05
const REFUEL_PCT_PER_MIN = 4
const REFUEL_BELOW_PCT = 10
const REFUEL_UP_TO_PCT = 95

// вероятности событий на одну сим-минуту
const SPEEDING_PER_MIN = 1 / 400
const SILENCE_PER_MIN = 1 / 1500

type Mode = 'drive' | 'unload' | 'refuel' | 'service' | 'parked'

interface Truck {
  vehicle: Vehicle
  deviceId: number
  out: Route
  back: Route
  route: Route
  km: number
  speed: number
  cruise: number
  fuel: number
  mode: Mode
  modeUntil: number
  speedingUntil: number
  speedingTarget: number
  silentUntil: number
  lastSentAt: number
  sentPos: LngLat
  nextEmitReal: number
  tripId: number | null
}

export interface MockOptions {
  seed?: number
  startAt?: number
  scale?: number
  // manual: время двигает только advance(), без таймера — для тестов
  manual?: boolean
}

const iso = (ms: number) => new Date(ms).toISOString()
const round = (n: number, digits: number) => Math.round(n * 10 ** digits) / 10 ** digits

export class MockFleetSource implements FleetSource {
  readonly sim: SimControls

  private readonly rng: Rng
  private readonly manual: boolean
  private simNow: number
  private realNow = 0
  private scale: number

  private readonly drivers: Driver[]
  private readonly rules: Record<AlertType, AlertRule>
  private readonly trucks: Truck[] = []
  private readonly trips = new Map<number, Trip>()
  private readonly alerts = new Map<number, Alert>()
  // открытые алерты по ключу «машина:тип» — не больше одного, как в stateful alert engine
  private readonly openAlerts = new Map<string, Alert>()
  private readonly scheduled: Array<{ at: number; run: () => void }> = []

  private readonly listeners = new Set<(event: FleetEvent) => void>()
  private timer: ReturnType<typeof setInterval> | null = null
  private lastTick = 0

  private nextTelemetryId = 50_000
  private nextTripId = 300
  private nextAlertId = 1_000

  constructor(opts: MockOptions = {}) {
    this.rng = createRng(opts.seed ?? 7)
    this.simNow = opts.startAt ?? Date.now()
    this.scale = opts.scale ?? 60
    this.manual = opts.manual ?? false

    const createdAt = iso(this.simNow - 90 * 24 * HOUR)
    this.drivers = makeDrivers(createdAt)
    this.rules = Object.fromEntries(makeRules(createdAt).map((r) => [r.type, r])) as Record<AlertType, AlertRule>

    this.seedFleet()
    this.seedHistory()
    this.scheduleOpening()

    this.sim = {
      now: () => this.simNow,
      scale: () => this.scale,
      setScale: (scale) => {
        this.scale = Math.max(0, scale)
      },
    }
  }

  async load(): Promise<FleetSnapshot> {
    return structuredClone({
      org: ORG,
      vehicles: this.trucks.map((t) => t.vehicle),
      drivers: this.drivers,
      trips: [...this.trips.values()],
      positions: this.trucks.map((t) => this.telemetryOf(t, 0)),
      alerts: [...this.alerts.values()],
      rules: Object.values(this.rules),
    })
  }

  subscribe({ onEvent }: SubscribeHandlers): () => void {
    this.listeners.add(onEvent)
    if (!this.manual && !this.timer) {
      this.lastTick = performance.now()
      this.timer = setInterval(() => {
        const now = performance.now()
        // во фоновой вкладке таймер засыпает — не прыгаем на минуты вперёд
        this.advance(Math.min(now - this.lastTick, 1000))
        this.lastTick = now
      }, TICK_MS)
    }
    return () => {
      this.listeners.delete(onEvent)
      if (this.listeners.size === 0 && this.timer) {
        clearInterval(this.timer)
        this.timer = null
      }
    }
  }

  async acknowledgeAlert(alertId: number): Promise<void> {
    const alert = this.alerts.get(alertId)
    if (!alert) throw new Error(`алерт ${alertId} не найден`)
    if (alert.status !== 'FIRED') return
    alert.status = 'ACKNOWLEDGED'
    alert.acknowledged_at = iso(this.simNow)
    alert.acknowledged_by = CURRENT_USER_ID
    this.publish({ type: 'alert', data: { ...alert } })
  }

  // advance двигает симуляцию на realMs реального времени
  advance(realMs: number): void {
    if (this.scale === 0) return
    for (let left = realMs; left > 0; left -= TICK_MS) {
      this.step(Math.min(TICK_MS, left))
    }
  }

  private step(realMs: number): void {
    this.realNow += realMs
    const dt = realMs * this.scale
    this.simNow += dt

    for (let i = this.scheduled.length - 1; i >= 0; i--) {
      if (this.scheduled[i].at <= this.simNow) this.scheduled.splice(i, 1)[0].run()
    }
    for (const truck of this.trucks) this.move(truck, dt)
    for (const truck of this.trucks) {
      if (this.realNow < truck.nextEmitReal) continue
      truck.nextEmitReal += EMIT_EVERY_MS
      if (this.simNow >= truck.silentUntil) this.emitTelemetry(truck)
    }
    this.watchdog()
  }

  private move(truck: Truck, dt: number): void {
    const dtMin = dt / MIN

    if (truck.mode !== 'drive') {
      truck.speed = 0
      if (truck.mode === 'refuel') {
        truck.fuel = Math.min(REFUEL_UP_TO_PCT, truck.fuel + REFUEL_PCT_PER_MIN * dtMin)
        if (truck.fuel >= REFUEL_UP_TO_PCT) truck.mode = 'drive'
        return
      }
      if (this.simNow < truck.modeUntil) return
      if (truck.mode === 'service') {
        this.setVehicleStatus(truck, 'IDLE')
        truck.mode = 'parked'
        truck.modeUntil = this.simNow + this.rng.range(10, 25) * MIN
      } else {
        this.startTrip(truck)
      }
      return
    }

    if (this.simNow >= truck.speedingUntil && this.rng.chance(SPEEDING_PER_MIN * dtMin)) {
      truck.speedingUntil = this.simNow + this.rng.range(5, 11) * MIN
      truck.speedingTarget = this.rng.range(96, 109)
    }
    if (this.simNow >= truck.silentUntil && this.rng.chance(SILENCE_PER_MIN * dtMin)) {
      truck.silentUntil = this.simNow + this.rng.range(6, 14) * MIN
    }

    const target = this.simNow < truck.speedingUntil ? truck.speedingTarget : truck.cruise
    truck.speed += (target - truck.speed) * Math.min(1, dtMin * 1.5)

    const km = (truck.speed * dt) / HOUR
    truck.km += km
    truck.fuel = Math.max(0, truck.fuel - km * FUEL_PCT_PER_KM)

    if (truck.fuel < REFUEL_BELOW_PCT) {
      truck.mode = 'refuel'
      truck.speedingUntil = 0
    }
    if (truck.km >= truck.route.lengthKm) this.completeTrip(truck)
  }

  private emitTelemetry(truck: Truck): void {
    const pos = pointAlong(truck.route, truck.km).pos
    const tel = this.telemetryOf(truck, haversineKm(truck.sentPos, pos))
    tel.telemetry_id = ++this.nextTelemetryId
    truck.sentPos = pos
    truck.lastSentAt = this.simNow
    this.publish({ type: 'telemetry', data: tel })

    const trip = truck.tripId != null ? this.trips.get(truck.tripId) : undefined
    if (trip) {
      trip.distance_km = round(trip.distance_km + tel.distance_km, 2)
      trip.telemetry_count += 1
      trip.max_speed_kmh = Math.max(trip.max_speed_kmh, tel.speed_kmh)
      const hours = (this.simNow - Date.parse(trip.started_at)) / HOUR
      trip.avg_speed_kmh = hours > 0 ? round(trip.distance_km / hours, 1) : 0
      this.publish({ type: 'trip', data: { ...trip } })
    }

    this.evaluate(truck, tel)
  }

  // evaluate — правила по свежей точке, как AlertService: открыть, если нарушение, закрыть, если норма
  private evaluate(truck: Truck, tel: TelemetryEvent): void {
    const vid = truck.vehicle.id
    const speed = this.rules.SPEED_EXCEEDED
    if (tel.speed_kmh > speed.threshold) {
      this.fire(vid, speed, tel.speed_kmh, `${Math.round(tel.speed_kmh)} км/ч при пороге ${speed.threshold}`)
    } else {
      this.resolve(vid, 'SPEED_EXCEEDED')
    }

    const fuel = this.rules.LOW_FUEL
    if (tel.fuel != null && tel.fuel < fuel.threshold) {
      this.fire(vid, fuel, tel.fuel, `Осталось ${Math.round(tel.fuel)}% при пороге ${fuel.threshold}%`)
    } else {
      this.resolve(vid, 'LOW_FUEL')
    }

    // трекер снова на связи — инцидент закрывается сам, как в OfflineWorker
    this.resolve(vid, 'DEVICE_OFFLINE')
  }

  private watchdog(): void {
    const rule = this.rules.DEVICE_OFFLINE
    for (const truck of this.trucks) {
      const silentMin = (this.simNow - truck.lastSentAt) / MIN
      if (silentMin > rule.threshold) {
        this.fire(truck.vehicle.id, rule, round(silentMin, 0), `Трекер молчит ${Math.round(silentMin)} мин`)
      }
    }
  }

  private fire(vehicleId: number, rule: AlertRule, value: number, message: string): void {
    const key = `${vehicleId}:${rule.type}`
    if (this.openAlerts.has(key)) return
    const alert: Alert = {
      id: ++this.nextAlertId,
      vehicle_id: vehicleId,
      rule_id: rule.id,
      type: rule.type,
      message,
      severity: rule.severity,
      value: round(value, 1),
      status: 'FIRED',
      created_at: iso(this.simNow),
      acknowledged_at: null,
      acknowledged_by: null,
      resolved_at: null,
      resolved_by: null,
    }
    this.alerts.set(alert.id, alert)
    this.openAlerts.set(key, alert)
    this.publish({ type: 'alert', data: { ...alert } })
  }

  private resolve(vehicleId: number, type: AlertType): void {
    const key = `${vehicleId}:${type}`
    const alert = this.openAlerts.get(key)
    if (!alert) return
    this.openAlerts.delete(key)
    alert.status = 'RESOLVED'
    alert.resolved_at = iso(this.simNow)
    this.publish({ type: 'alert', data: { ...alert } })
  }

  private startTrip(truck: Truck): void {
    const busy = new Set([...this.trips.values()].filter((t) => t.status === 'RUNNING').map((t) => t.driver_id))
    const free = this.drivers.filter((d) => !busy.has(d.id))
    if (free.length === 0) {
      truck.mode = 'parked'
      truck.modeUntil = this.simNow + 10 * MIN
      return
    }
    const trip = this.openTrip(truck, this.rng.pick(free).id, this.simNow, 0)
    this.publish({ type: 'trip', data: { ...trip } })
    this.setVehicleStatus(truck, 'ON_TRIP')
  }

  private openTrip(truck: Truck, driverId: number, startedAt: number, distanceKm: number): Trip {
    const trip: Trip = {
      id: ++this.nextTripId,
      driver_id: driverId,
      vehicle_id: truck.vehicle.id,
      started_at: iso(startedAt),
      ended_at: null,
      status: 'RUNNING',
      distance_km: round(distanceKm, 2),
      avg_speed_kmh: 0,
      max_speed_kmh: 0,
      telemetry_count: 0,
    }
    this.trips.set(trip.id, trip)
    truck.tripId = trip.id
    truck.mode = 'drive'
    truck.cruise = this.rng.range(68, 86)
    truck.vehicle.status = 'ON_TRIP'
    return trip
  }

  private completeTrip(truck: Truck): void {
    const trip = truck.tripId != null ? this.trips.get(truck.tripId) : undefined
    if (trip) {
      trip.status = 'SUCCEEDED'
      trip.ended_at = iso(this.simNow)
      this.publish({ type: 'trip', data: { ...trip } })
    }
    truck.tripId = null
    truck.speed = 0
    truck.speedingUntil = 0
    // разворот: конец маршрута становится началом обратного
    truck.route = truck.route === truck.out ? truck.back : truck.out
    truck.km = 0
    truck.mode = 'unload'
    truck.modeUntil = this.simNow + this.rng.range(20, 45) * MIN
    this.setVehicleStatus(truck, 'IDLE')
  }

  private setVehicleStatus(truck: Truck, status: VehicleStatus): void {
    truck.vehicle.status = status
    this.publish({ type: 'vehicle', data: { ...truck.vehicle } })
  }

  private telemetryOf(truck: Truck, distanceKm: number): TelemetryEvent {
    const pos = pointAlong(truck.route, truck.km).pos
    return {
      telemetry_id: this.nextTelemetryId,
      vehicle_id: truck.vehicle.id,
      device_id: truck.deviceId,
      lat: round(pos[1], 6),
      lon: round(pos[0], 6),
      fuel: round(truck.fuel, 1),
      received_at: iso(this.simNow),
      trip_id: truck.tripId ?? 0,
      distance_km: round(distanceKm, 3),
      speed_kmh: round(truck.speed, 1),
    }
  }

  private publish(event: FleetEvent): void {
    for (const listener of this.listeners) listener(event)
  }

  // seedFleet расставляет машины: большинство в рейсе, две на стоянке, одна на ТО
  private seedFleet(): void {
    const routes = CORRIDORS.map((c) => {
      const path = corridorPath(c)
      return { out: buildRoute(path), back: buildRoute([...path].reverse()) }
    })
    const vehicles = makeVehicles(this.rng, VEHICLE_COUNT)
    const freeDrivers = [...this.drivers]

    vehicles.forEach((vehicle, i) => {
      const { out, back } = routes[i % routes.length]
      const driving = i < DRIVING_COUNT
      const route = driving && this.rng.chance(0.5) ? back : out
      const truck: Truck = {
        vehicle,
        deviceId: 100 + vehicle.id,
        out,
        back,
        route,
        km: driving ? this.rng.range(0.1, 0.85) * route.lengthKm : 0,
        speed: 0,
        cruise: 0,
        fuel: this.rng.range(30, 92),
        mode: 'parked',
        modeUntil: this.simNow + this.rng.range(8, 30) * MIN,
        speedingUntil: 0,
        speedingTarget: 0,
        silentUntil: 0,
        lastSentAt: this.simNow,
        sentPos: pointAlong(route, 0).pos,
        nextEmitReal: this.rng.range(0, EMIT_EVERY_MS),
        tripId: null,
      }
      truck.sentPos = pointAlong(route, truck.km).pos

      if (driving) {
        const driver = freeDrivers.splice(this.rng.int(0, freeDrivers.length - 1), 1)[0]
        const avg = this.rng.range(70, 78)
        const trip = this.openTrip(truck, driver.id, this.simNow - (truck.km / avg) * HOUR, truck.km)
        truck.speed = truck.cruise
        trip.avg_speed_kmh = round(avg, 1)
        trip.max_speed_kmh = round(truck.cruise + this.rng.range(2, 8), 1)
        trip.telemetry_count = Math.round((truck.km / avg) * 360)
      } else if (i === VEHICLE_COUNT - 1) {
        truck.mode = 'service'
        truck.modeUntil = this.simNow + this.rng.range(30, 50) * MIN
        vehicle.status = 'IN_SERVICE'
      }
      this.trucks.push(truck)
    })

    // одна машина почти на нуле — LOW_FUEL сработает в первые секунды
    this.trucks[4].fuel = 15.8
  }

  // seedHistory добавляет завершённые за смену рейсы, чтобы рейтинг водителей не был пустым
  private seedHistory(): void {
    const busy = new Map<number, Array<[number, number]>>()
    const overlaps = (vid: number, from: number, to: number) =>
      (busy.get(vid) ?? []).some(([a, b]) => from < b && to > a)
    const reserve = (vid: number, from: number, to: number) =>
      busy.set(vid, [...(busy.get(vid) ?? []), [from, to]])

    const running = [...this.trips.values()]
    for (const t of running) reserve(t.vehicle_id, Date.parse(t.started_at), Infinity)

    const shiftStart = this.simNow - 16 * HOUR
    const history: Trip[] = []
    for (const driver of this.drivers) {
      const current = running.find((t) => t.driver_id === driver.id)
      let cursor = (current ? Date.parse(current.started_at) : this.simNow) - this.rng.range(0.5, 1.5) * HOUR
      for (let n = this.rng.int(1, 2); n > 0; n--) {
        const hours = this.rng.range(3, 7)
        const from = cursor - hours * HOUR
        if (from < shiftStart) break
        const vehicleId = this.rng.int(1, VEHICLE_COUNT)
        if (overlaps(vehicleId, from, cursor)) continue
        reserve(vehicleId, from, cursor)

        const avg = this.rng.range(68, 80)
        history.push({
          id: 0,
          driver_id: driver.id,
          vehicle_id: vehicleId,
          started_at: iso(from),
          ended_at: iso(cursor),
          status: 'SUCCEEDED',
          distance_km: round(avg * hours, 2),
          avg_speed_kmh: round(avg, 1),
          max_speed_kmh: round(avg + this.rng.range(8, 30), 1),
          telemetry_count: Math.round(hours * 360),
        })
        cursor = from - this.rng.range(0.5, 2) * HOUR
      }
    }

    history.sort((a, b) => Date.parse(a.started_at) - Date.parse(b.started_at))
    const speed = this.rules.SPEED_EXCEEDED
    let tripId = 100
    let alertId = 500
    for (const trip of history) {
      trip.id = ++tripId
      this.trips.set(trip.id, trip)
      if (trip.max_speed_kmh <= speed.threshold) continue
      const from = Date.parse(trip.started_at)
      const at = from + this.rng.range(0.2, 0.8) * (Date.parse(trip.ended_at!) - from)
      this.alerts.set(++alertId, {
        id: alertId,
        vehicle_id: trip.vehicle_id,
        rule_id: speed.id,
        type: 'SPEED_EXCEEDED',
        message: `${Math.round(trip.max_speed_kmh)} км/ч при пороге ${speed.threshold}`,
        severity: speed.severity,
        value: trip.max_speed_kmh,
        status: 'RESOLVED',
        created_at: iso(at),
        acknowledged_at: null,
        acknowledged_by: null,
        resolved_at: iso(at + this.rng.range(3, 9) * MIN),
        resolved_by: null,
      })
    }
  }

  // scheduleOpening — гарантированные события в первые секунды, чтобы экран не был пустым
  private scheduleOpening(): void {
    const speeder = this.trucks[2]
    const silent = this.trucks[6]
    this.scheduled.push(
      {
        at: this.simNow + 3 * MIN,
        run: () => {
          speeder.speedingUntil = this.simNow + 8 * MIN
          speeder.speedingTarget = 104
        },
      },
      { at: this.simNow + 12 * MIN, run: () => (silent.silentUntil = this.simNow + 11 * MIN) },
    )
  }
}
