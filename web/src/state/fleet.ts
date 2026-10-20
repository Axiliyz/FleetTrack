import { create } from 'zustand'
import type { LinkState } from '../source/FleetSource'
import type { Alert, AlertRule, Driver, FleetEvent, FleetSnapshot, Org, TelemetryEvent, Trip, Vehicle } from '../types'
import { ALERT_LABEL, severityRank } from './selectors'

export type Tone = 'alarm' | 'warn' | 'ok' | 'info'

export interface LogEntry {
  key: string
  at: string
  tone: Tone
  text: string
  vehicleId: number
  alertId?: number
}

const LOG_LIMIT = 80

interface FleetState {
  status: 'loading' | 'ready' | 'error'
  error: string | null
  // состояние живого потока; null — симулятор, у него связи с сервером нет
  link: LinkState | null
  org: Org | null
  vehicles: Record<number, Vehicle>
  drivers: Record<number, Driver>
  trips: Record<number, Trip>
  // последняя точка по каждой машине
  telemetry: Record<number, TelemetryEvent>
  alerts: Record<number, Alert>
  rules: AlertRule[]
  // время по последней точке: у симулятора оно ускорено
  clock: number
  log: LogEntry[]

  selectedId: number | null
  followId: number | null
  // запрос на перелёт камеры: seq растёт, чтобы повторный клик по той же машине тоже срабатывал
  camera: { vehicleId: number; seq: number } | null
  toastAlertId: number | null
  // null — живые данные без управления временем, 0 — пауза
  timeScale: number | null
  lastScale: number
  muted: boolean
  rosterOpen: boolean

  init(snapshot: FleetSnapshot, timeScale: number | null): void
  fail(message: string): void
  setLink(state: LinkState | null): void
  // reset возвращает стор к начальному состоянию: смена пользователя или источника
  reset(): void
  note(entry: Omit<LogEntry, 'key' | 'at'>): void
  apply(event: FleetEvent): void
  select(vehicleId: number | null, opts?: { fly?: boolean }): void
  setFollow(vehicleId: number | null): void
  setScale(scale: number): void
  showToast(alertId: number): void
  dismissToast(): void
  toggleMuted(): void
  setRosterOpen(open: boolean): void
}

const byId = <T extends { id: number }>(items: T[]) =>
  Object.fromEntries(items.map((item) => [item.id, item])) as Record<number, T>

const toneOf = (alert: Alert): Tone => {
  const rank = severityRank(alert.severity)
  return rank >= 2 ? 'alarm' : rank === 1 ? 'warn' : 'info'
}

const initialState = {
  status: 'loading' as FleetState['status'],
  error: null as string | null,
  link: null as LinkState | null,
  org: null as Org | null,
  vehicles: {} as Record<number, Vehicle>,
  drivers: {} as Record<number, Driver>,
  trips: {} as Record<number, Trip>,
  telemetry: {} as Record<number, TelemetryEvent>,
  alerts: {} as Record<number, Alert>,
  rules: [] as AlertRule[],
  clock: Date.now(),
  log: [] as LogEntry[],

  selectedId: null as number | null,
  followId: null as number | null,
  camera: null as FleetState['camera'],
  toastAlertId: null as number | null,
  timeScale: null as number | null,
  lastScale: 60,
  muted: false,
  rosterOpen: false,
}

export const useFleet = create<FleetState>()((set, get) => ({
  ...initialState,

  reset() {
    // звук и раскладку панелей диспетчер выбирал сам — их не сбрасываем
    const { muted, rosterOpen } = get()
    set({ ...initialState, clock: Date.now(), muted, rosterOpen })
  },

  init(snapshot, timeScale) {
    const telemetry: Record<number, TelemetryEvent> = {}
    for (const point of snapshot.positions) telemetry[point.vehicle_id] = point
    const clock = Math.max(0, ...snapshot.positions.map((p) => Date.parse(p.received_at)))
    // повторный снимок после переподключения заменяет данные, но не журнал и не выбор диспетчера
    const again = get().status === 'ready'
    const vehicles = byId(snapshot.vehicles)
    const selectedId = get().selectedId
    set({
      status: 'ready',
      error: null,
      org: snapshot.org,
      vehicles,
      drivers: byId(snapshot.drivers),
      trips: byId(snapshot.trips),
      telemetry,
      alerts: byId(snapshot.alerts),
      rules: snapshot.rules,
      clock: clock || Date.now(),
      log: again ? get().log : [],
      selectedId: selectedId !== null && vehicles[selectedId] ? selectedId : null,
      timeScale,
      lastScale: timeScale || get().lastScale,
    })
  },

  setLink(state) {
    set({ link: state })
  },

  note(entry) {
    const at = new Date(get().clock).toISOString()
    set((s) => ({ log: [{ ...entry, key: `note-${Date.now()}-${Math.random()}`, at }, ...s.log].slice(0, LOG_LIMIT) }))
  },

  fail(message) {
    set({ status: 'error', error: message })
  },

  apply(event) {
    const state = get()
    switch (event.type) {
      case 'telemetry': {
        const point = event.data
        set({
          telemetry: { ...state.telemetry, [point.vehicle_id]: point },
          clock: Math.max(state.clock, Date.parse(point.received_at)),
        })
        return
      }
      case 'vehicle': {
        const vehicle = event.data
        if (vehicle.status === 'DELETED') {
          // удалённая машина уходит с карты вместе с последней точкой
          const vehicles = { ...state.vehicles }
          const telemetry = { ...state.telemetry }
          delete vehicles[vehicle.id]
          delete telemetry[vehicle.id]
          set({
            vehicles,
            telemetry,
            ...(state.selectedId === vehicle.id && { selectedId: null, followId: null }),
          })
          return
        }
        set({ vehicles: { ...state.vehicles, [vehicle.id]: vehicle } })
        return
      }
      case 'trip': {
        const trip = event.data
        const prev = state.trips[trip.id]
        let entry: LogEntry | null = null
        if (!prev && trip.status === 'RUNNING') {
          const driver = state.drivers[trip.driver_id]?.name ?? `водитель №${trip.driver_id}`
          entry = { key: `trip-${trip.id}-start`, at: trip.started_at, tone: 'info', text: `Рейс №${trip.id} начат, за рулём ${driver}`, vehicleId: trip.vehicle_id }
        } else if (prev?.status === 'RUNNING' && trip.status === 'SUCCEEDED') {
          entry = { key: `trip-${trip.id}-end`, at: trip.ended_at ?? trip.started_at, tone: 'ok', text: `Рейс №${trip.id} завершён, ${Math.round(trip.distance_km)} км`, vehicleId: trip.vehicle_id }
        }
        set({
          trips: { ...state.trips, [trip.id]: trip },
          ...(entry && { log: [entry, ...state.log].slice(0, LOG_LIMIT) }),
        })
        return
      }
      case 'alert': {
        const alert = event.data
        const prev = state.alerts[alert.id]
        const label = ALERT_LABEL[alert.type]
        let entry: LogEntry | null = null
        if (!prev) {
          entry = { key: `alert-${alert.id}`, at: alert.created_at, tone: toneOf(alert), text: `${label}: ${alert.message}`, vehicleId: alert.vehicle_id, alertId: alert.id }
        } else if (prev.status !== alert.status && alert.status === 'ACKNOWLEDGED') {
          entry = { key: `alert-${alert.id}-ack`, at: alert.acknowledged_at ?? alert.created_at, tone: 'info', text: `Принято в работу: ${label.toLowerCase()}`, vehicleId: alert.vehicle_id }
        } else if (prev.status !== alert.status && alert.status === 'RESOLVED') {
          entry = { key: `alert-${alert.id}-resolved`, at: alert.resolved_at ?? alert.created_at, tone: 'ok', text: `Закрыто: ${label.toLowerCase()}`, vehicleId: alert.vehicle_id }
        }
        set({
          alerts: { ...state.alerts, [alert.id]: alert },
          ...(entry && { log: [entry, ...state.log].slice(0, LOG_LIMIT) }),
          ...(alert.status !== 'FIRED' && state.toastAlertId === alert.id && { toastAlertId: null }),
        })
        return
      }
    }
  },

  select(vehicleId, opts) {
    const { followId, camera } = get()
    set({
      selectedId: vehicleId,
      followId: vehicleId === null || followId !== vehicleId ? null : followId,
      camera: opts?.fly && vehicleId !== null ? { vehicleId, seq: (camera?.seq ?? 0) + 1 } : camera,
    })
  },

  setFollow(vehicleId) {
    set({ followId: vehicleId })
  },

  setScale(scale) {
    set((s) => ({ timeScale: scale, lastScale: scale > 0 ? scale : s.lastScale }))
  },

  showToast(alertId) {
    set({ toastAlertId: alertId })
  },

  dismissToast() {
    set({ toastAlertId: null })
  },

  toggleMuted() {
    set((s) => ({ muted: !s.muted }))
  },

  setRosterOpen(open) {
    set({ rosterOpen: open })
  },
}))
