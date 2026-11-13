// Контракт фронта с бэком. Имена полей повторяют JSON из internal/handler/dto и internal/model.

// DELETED приходит только событием об удалении: в сторе таких машин не бывает
export type VehicleStatus = 'IDLE' | 'ON_TRIP' | 'IN_SERVICE' | 'DELETED'

// Vehicle — dto.VehicleResponse
export interface Vehicle {
  id: number
  organization_id: number
  vin: string
  number_plate: string
  model: string
  status: VehicleStatus
}

// Driver — dto.DriverResponse
export interface Driver {
  id: number
  organization_id: number
  name: string
  created_at: string
  updated_at: string | null
}

export type TripStatus = 'RUNNING' | 'CANCELLED' | 'SUCCEEDED' | 'SLEEPING' | 'SERVING'

// Trip — model.Trip
export interface Trip {
  id: number
  driver_id: number
  vehicle_id: number
  started_at: string
  ended_at: string | null
  status: TripStatus
  distance_km: number
  avg_speed_kmh: number
  max_speed_kmh: number
  telemetry_count: number
}

// TelemetryEvent — точка телеметрии, dto.TelemetryResponse
export interface TelemetryEvent {
  telemetry_id: number
  vehicle_id: number
  device_id: number
  lat: number
  lon: number
  fuel: number | null
  received_at: string
  trip_id: number
  distance_km: number
  speed_kmh: number
}

export type AlertType = 'SPEED_EXCEEDED' | 'DEVICE_OFFLINE' | 'LOW_FUEL'
export type Severity = 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL'
export type AlertStatus = 'FIRED' | 'ACKNOWLEDGED' | 'RESOLVED'

// Alert — dto.AlertResponse (GET /alerts, PATCH /alerts/{id})
export interface Alert {
  id: number
  vehicle_id: number
  rule_id: number
  type: AlertType
  message: string
  severity: Severity
  value: number | null
  status: AlertStatus
  created_at: string
  acknowledged_at: string | null
  acknowledged_by: number | null
  resolved_at: string | null
  resolved_by: number | null
}

// AlertRule — dto.AlertRuleResponse
export interface AlertRule {
  id: number
  organization_id: number
  type: AlertType
  name: string
  threshold: number
  severity: Severity
  enabled: boolean
  created_at: string
}

// Org — dto.OrgResponse
export interface Org {
  id: number
  name: string
}

// FleetSnapshot — dto.FleetSnapshotResponse (GET /fleet/snapshot)
export interface FleetSnapshot {
  org: Org
  vehicles: Vehicle[]
  drivers: Driver[]
  // активные рейсы и рейсы за текущую смену: из них считается рейтинг водителей
  trips: Trip[]
  // последняя точка по каждой машине
  positions: TelemetryEvent[]
  // открытые алерты и закрытые за смену
  alerts: Alert[]
  rules: AlertRule[]
}

export type DeviceStatus = 'ACTIVE' | 'INACTIVE' | 'MAINTENANCE'

// Device — трекер, dto.DeviceResponse
export interface Device {
  id: number
  serial_number: string
  status: DeviceStatus
  created_at: string
  organization_id: number
}

// Assignment — привязка трекера к машине, dto.AssignmentResponse
export interface Assignment {
  id: number
  device_id: number
  vehicle_id: number
  started_at: string
  ended_at?: string
}

export type UserRole = 'ADMIN' | 'DISPATCHER' | 'ANALYTIC' | 'DRIVER'

// User — dto.UserResponse
export interface User {
  id: number
  organization_id: number
  name: string
  email: string
  role: UserRole
}

// Claims — полезная нагрузка access-токена, model.JWTClaims
export interface Claims {
  user_id: number
  organization_id: number
  driver_id: number | null
  role: UserRole
  exp: number
}

// FleetEvent — сообщение из потока SSE GET /stream; data — тот же JSON, что в REST-ответах
export type FleetEvent =
  | { type: 'telemetry'; data: TelemetryEvent }
  | { type: 'alert'; data: Alert }
  | { type: 'trip'; data: Trip }
  | { type: 'vehicle'; data: Vehicle }
