import { useQuery } from '@tanstack/react-query'
import { api } from '../../api/client'
import { useApiMutation } from '../../api/mutations'
import { LIVE } from '../../config'
import type { Assignment, Device } from '../../types'
import { telemetryKeys } from '../telemetry/api'

export const devicesApi = {
  get: (id: number) => api<Device>(`/devices/${id}`),
  create: (serialNumber: string) =>
    api<Device>('/devices', { method: 'POST', body: JSON.stringify({ serial_number: serialNumber }) }),
  remove: (id: number) => api<Device>(`/devices/${id}`, { method: 'DELETE' }),
  assign: (deviceId: number, vehicleId: number) =>
    api<Assignment>('/assignments', {
      method: 'POST',
      body: JSON.stringify({ device_id: deviceId, vehicle_id: vehicleId }),
    }),
}

export const deviceKeys = {
  all: ['devices'] as const,
  one: (id: number) => ['devices', 'one', id] as const,
}

export function useDevice(id: number | null) {
  return useQuery({
    queryKey: deviceKeys.one(id ?? 0),
    queryFn: () => devicesApi.get(id ?? 0),
    enabled: LIVE && id !== null,
  })
}

export const useCreateDevice = () => useApiMutation(devicesApi.create, [deviceKeys.all])
// удалённый трекер пропадает и из «на связи»
export const useDeleteDevice = () => useApiMutation(devicesApi.remove, [deviceKeys.all, telemetryKeys.recent])
export const useAssignDevice = () =>
  useApiMutation(({ deviceId, vehicleId }: { deviceId: number; vehicleId: number }) =>
    devicesApi.assign(deviceId, vehicleId), [deviceKeys.all])
