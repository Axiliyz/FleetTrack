// Подписи перечислений бэка. Тревоги и состояния ламп подписаны в state/selectors.ts
import type { Tone } from '../state/fleet'
import type { AlertStatus, DeviceStatus, Severity, TripStatus, UserRole, VehicleStatus } from '../types'

// optionsOf превращает подписи в варианты для <select>; keys — подмножество и порядок
export function optionsOf<K extends string>(labels: Record<K, string>, keys = Object.keys(labels) as K[]) {
  return keys.map((value) => ({ value, label: labels[value] }))
}

export const VEHICLE_STATUS_LABEL: Record<VehicleStatus, string> = {
  IDLE: 'Свободна',
  ON_TRIP: 'В рейсе',
  IN_SERVICE: 'На обслуживании',
  DELETED: 'Удалена',
}

// цвет лампы статуса машины — тот же, что на карте
export const VEHICLE_STATUS_COND: Record<VehicleStatus, string> = {
  IDLE: 'idle',
  ON_TRIP: 'trip',
  IN_SERVICE: 'service',
  DELETED: 'offline',
}

export const TRIP_STATUS_LABEL: Record<TripStatus, string> = {
  RUNNING: 'В пути',
  SLEEPING: 'Ожидает',
  SERVING: 'На ТО',
  SUCCEEDED: 'Завершён',
  CANCELLED: 'Отменён',
}

export const TRIP_STATUS_TONE: Record<TripStatus, Tone> = {
  RUNNING: 'warn',
  SLEEPING: 'info',
  SERVING: 'info',
  SUCCEEDED: 'ok',
  CANCELLED: 'info',
}

export const SEVERITY_LABEL: Record<Severity, string> = {
  LOW: 'Низкая',
  MEDIUM: 'Средняя',
  HIGH: 'Высокая',
  CRITICAL: 'Критическая',
}

export const SEVERITY_TONE: Record<Severity, Tone> = {
  LOW: 'info',
  MEDIUM: 'warn',
  HIGH: 'alarm',
  CRITICAL: 'alarm',
}

export const ALERT_STATUS_LABEL: Record<AlertStatus, string> = {
  FIRED: 'Не принята',
  ACKNOWLEDGED: 'В работе',
  RESOLVED: 'Закрыта',
}

export const ALERT_STATUS_TONE: Record<AlertStatus, Tone> = {
  FIRED: 'alarm',
  ACKNOWLEDGED: 'warn',
  RESOLVED: 'ok',
}

export const DEVICE_STATUS_LABEL: Record<DeviceStatus, string> = {
  ACTIVE: 'Работает',
  INACTIVE: 'Отключён',
  MAINTENANCE: 'В ремонте',
}

export const ROLE_LABEL: Record<UserRole, string> = {
  ADMIN: 'Администратор',
  DISPATCHER: 'Диспетчер',
  ANALYTIC: 'Аналитик',
  DRIVER: 'Водитель',
}

export const ROLE_NOTE: Record<UserRole, string> = {
  ADMIN: 'всё, включая пользователей',
  DISPATCHER: 'карта, парк, рейсы, тревоги',
  ANALYTIC: 'карта и данные только для чтения',
  DRIVER: 'только свои рейсы',
}
