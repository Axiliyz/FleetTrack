import { useQuery } from '@tanstack/react-query'
import { api } from '../../api/client'
import { useApiMutation } from '../../api/mutations'
import { LOOKUP_LIMIT, apiList } from '../../api/query'
import type { TelemetryEvent } from '../../types'
import { tripKeys } from '../trips/api'

// TrackScope — чей трек показываем: машины за период или одного рейса целиком
export type TrackScope = { vehicleId: number; from: string } | { tripId: number }

// NewPoint — dto.TelemetryRequest: точка «как от терминала», время терминала ставит сервер
export interface NewPoint {
  device_id: number
  vehicle_id: number
  lat: number
  lon: number
  fuel: number | null
}

export const telemetryApi = {
  // бэк отдаёт точки от новых к старым; потолок — 500 штук, для трека этого хватает
  track: (scope: TrackScope) =>
    apiList<TelemetryEvent>(
      '/telemetry',
      'tripId' in scope
        ? { trip_id: scope.tripId, limit: LOOKUP_LIMIT }
        : { vehicle_id: scope.vehicleId, from: scope.from, limit: LOOKUP_LIMIT },
    ),
  recent: () => apiList<TelemetryEvent>('/telemetry', { limit: LOOKUP_LIMIT }),
  send: (point: NewPoint) => api<TelemetryEvent>('/telemetry', { method: 'POST', body: JSON.stringify(point) }),
  remove: (id: number) => api<TelemetryEvent>(`/telemetry/${id}`, { method: 'DELETE' }),
  clearVehicle: async (vehicleId: number) =>
    (await api<TelemetryEvent[] | null>(`/telemetry/vehicles/${vehicleId}`, { method: 'DELETE' })) ?? [],
}

export const telemetryKeys = {
  all: ['telemetry'] as const,
  track: (scope: TrackScope) => ['telemetry', 'track', scope] as const,
  recent: ['telemetry', 'recent'] as const,
}

export function useTrack(scope: TrackScope) {
  return useQuery({
    queryKey: telemetryKeys.track(scope),
    queryFn: () => telemetryApi.track(scope),
    // в хронологическом порядке: так строятся линия трека и график
    select: (points) => points.toReversed(),
    refetchInterval: 15_000,
  })
}

// useRecentDevices — трекеры, приславшие точки последними. Списка трекеров в API нет,
// поэтому «кто сейчас на связи» видно только по телеметрии
export function useRecentDevices() {
  return useQuery({
    queryKey: telemetryKeys.recent,
    queryFn: telemetryApi.recent,
    select: (points) => {
      const seen = new Map<number, TelemetryEvent>()
      for (const p of points) if (!seen.has(p.device_id)) seen.set(p.device_id, p)
      return [...seen.values()]
    },
    refetchInterval: 15_000,
  })
}

// точка двигает пробег и скорость рейса — их тоже перечитываем
export const useSendPoint = () => useApiMutation(telemetryApi.send, [telemetryKeys.all, tripKeys.all])
export const useDeletePoint = () => useApiMutation(telemetryApi.remove, [telemetryKeys.all])
export const useClearVehicleTrack = () => useApiMutation(telemetryApi.clearVehicle, [telemetryKeys.all])
