import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { api } from '../../api/client'
import { UserFacingError } from '../../api/errors'
import { useApiMutation } from '../../api/mutations'
import { apiList, pageParams, toPage, type QueryParams } from '../../api/query'
import { fromInputDate } from '../../lib/format'
import type { Trip, TripStatus, VehicleStatus } from '../../types'
import { vehicleKeys, vehiclesApi } from '../vehicles/api'

export const TRIP_FILTER_KEYS = ['status', 'vehicle_id', 'driver_id', 'started_from', 'started_to'] as const
export type TripFilter = Record<(typeof TRIP_FILTER_KEYS)[number], string>

// tripParams: даты в адресе хранятся как в <input type="date">, бэку нужен RFC3339
export const tripParams = (filter: TripFilter) => ({
  ...filter,
  started_from: fromInputDate(filter.started_from),
  started_to: fromInputDate(filter.started_to, true),
})

export const tripsApi = {
  list: (params: QueryParams) => apiList<Trip>('/trips', params),
  get: (id: number) => api<Trip>(`/trips/${id}`),
  start: (driverId: number, vehicleId: number) =>
    api<Trip>('/trips', { method: 'POST', body: JSON.stringify({ driver_id: driverId, vehicle_id: vehicleId }) }),
  setStatus: (id: number, status: TripStatus) =>
    api<Trip>(`/trips/${id}`, { method: 'PATCH', body: JSON.stringify({ status }) }),
  remove: (id: number) => api<Trip>(`/trips/${id}`, { method: 'DELETE' }),
}

export const tripKeys = {
  all: ['trips'] as const,
  page: (filter: TripFilter, offset: number) => ['trips', 'page', filter, offset] as const,
  one: (id: number) => ['trips', 'one', id] as const,
}

export function useTripPage(filter: TripFilter, offset: number) {
  return useQuery({
    queryKey: tripKeys.page(filter, offset),
    queryFn: async () => toPage(await tripsApi.list({ ...tripParams(filter), ...pageParams(offset) })),
    placeholderData: keepPreviousData,
  })
}

export function useTrip(id: number) {
  return useQuery({ queryKey: tripKeys.one(id), queryFn: () => tripsApi.get(id) })
}

// Бэк не связывает статус машины с рейсом: лампу «в рейсе» на карте зажигает отдельный PATCH /vehicles,
// как это делает и генератор парка. Если второй запрос не прошёл, рейс уже изменён — повторять всю операцию
// нельзя (создался бы второй рейс), поэтому ошибка говорит, что поправить статус машины надо вручную
export class VehicleNotSynced extends UserFacingError {
  readonly trip: Trip

  constructor(trip: Trip) {
    super(`Рейс №${trip.id} сохранён, но статус машины не обновился. Поправьте его в карточке машины`)
    this.trip = trip
  }
}

async function syncVehicle(trip: Trip, status: VehicleStatus): Promise<void> {
  try {
    await vehiclesApi.update(trip.vehicle_id, { status })
  } catch {
    throw new VehicleNotSynced(trip)
  }
}

// рейс, который больше не едет, освобождает машину
const FINISHED: ReadonlySet<TripStatus> = new Set(['SUCCEEDED', 'CANCELLED'])

export async function startTrip(driverId: number, vehicleId: number): Promise<Trip> {
  const trip = await tripsApi.start(driverId, vehicleId)
  await syncVehicle(trip, 'ON_TRIP')
  return trip
}

// Статус до изменения берём с сервера, а не из списка на экране: список мог устареть,
// пока генератор или другой диспетчер завершали рейсы
export async function setTripStatus(id: number, status: TripStatus): Promise<Trip> {
  const before = await tripsApi.get(id)
  const updated = await tripsApi.setStatus(id, status)
  if (before.status === 'RUNNING' && FINISHED.has(status)) await syncVehicle(updated, 'IDLE')
  if (before.status !== 'RUNNING' && status === 'RUNNING') await syncVehicle(updated, 'ON_TRIP')
  return updated
}

export async function removeTrip(id: number): Promise<void> {
  const before = await tripsApi.get(id)
  await tripsApi.remove(id)
  if (before.status === 'RUNNING') await syncVehicle(before, 'IDLE')
}

export const useStartTrip = () =>
  useApiMutation(
    ({ driverId, vehicleId }: { driverId: number; vehicleId: number }) => startTrip(driverId, vehicleId),
    [tripKeys.all, vehicleKeys.all],
  )

export const useSetTripStatus = () =>
  useApiMutation(({ id, status }: { id: number; status: TripStatus }) => setTripStatus(id, status), [
    tripKeys.all,
    vehicleKeys.all,
  ])

export const useDeleteTrip = () => useApiMutation(removeTrip, [tripKeys.all, vehicleKeys.all])
