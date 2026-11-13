import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { api } from '../../api/client'
import { useApiMutation } from '../../api/mutations'
import { LOOKUP_LIMIT, apiList, pageParams, toPage, type QueryParams } from '../../api/query'
import { indexById } from '../../lib/collections'
import type { Vehicle, VehicleStatus } from '../../types'

export const VEHICLE_FILTER_KEYS = ['status', 'number_plate', 'model'] as const
export type VehicleFilter = Record<(typeof VEHICLE_FILTER_KEYS)[number], string>

export interface NewVehicle {
  vin: string
  number_plate: string
  model: string
}

// VehiclePatch — dto.UpdateVehicleRequest. Поля организации нет: неизвестное поле бэк отклоняет с 400.
// Статус вручную - только IDLE или IN_SERVICE и только у машины не в рейсе
export interface VehiclePatch {
  number_plate?: string
  status?: VehicleStatus
}

export const vehiclesApi = {
  list: (params: QueryParams) => apiList<Vehicle>('/vehicles', params),
  get: (id: number) => api<Vehicle>(`/vehicles/${id}`),
  create: (v: NewVehicle) => api<Vehicle>('/vehicles', { method: 'POST', body: JSON.stringify(v) }),
  update: (id: number, patch: VehiclePatch) =>
    api<Vehicle>(`/vehicles/${id}`, { method: 'PATCH', body: JSON.stringify(patch) }),
  remove: (id: number) => api<Vehicle>(`/vehicles/${id}`, { method: 'DELETE' }),
}

export const vehicleKeys = {
  all: ['vehicles'] as const,
  page: (filter: VehicleFilter, offset: number) => ['vehicles', 'page', filter, offset] as const,
  index: ['vehicles', 'index'] as const,
  one: (id: number) => ['vehicles', 'one', id] as const,
}

export function useVehiclePage(filter: VehicleFilter, offset: number) {
  return useQuery({
    queryKey: vehicleKeys.page(filter, offset),
    queryFn: async () => toPage(await vehiclesApi.list({ ...filter, ...pageParams(offset) })),
    placeholderData: keepPreviousData,
  })
}

// useVehicleIndex — все машины по id: номера в таблицах рейсов и тревог, выбор машины в формах
export function useVehicleIndex() {
  return useQuery({
    queryKey: vehicleKeys.index,
    queryFn: () => vehiclesApi.list({ limit: LOOKUP_LIMIT }),
    select: indexById,
  })
}

export function useVehicle(id: number) {
  return useQuery({ queryKey: vehicleKeys.one(id), queryFn: () => vehiclesApi.get(id) })
}

export const useCreateVehicle = () => useApiMutation(vehiclesApi.create, [vehicleKeys.all])

export const useUpdateVehicle = () =>
  useApiMutation(({ id, patch }: { id: number; patch: VehiclePatch }) => vehiclesApi.update(id, patch), [
    vehicleKeys.all,
  ])

export const useDeleteVehicle = () => useApiMutation(vehiclesApi.remove, [vehicleKeys.all])
