import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { api } from '../../api/client'
import { useApiMutation } from '../../api/mutations'
import { LOOKUP_LIMIT, apiList, pageParams, toPage, type QueryParams } from '../../api/query'
import { indexById } from '../../lib/collections'
import type { Driver } from '../../types'

export const DRIVER_FILTER_KEYS = ['name'] as const
export type DriverFilter = Record<(typeof DRIVER_FILTER_KEYS)[number], string>

export const driversApi = {
  list: (params: QueryParams) => apiList<Driver>('/drivers', params),
  get: (id: number) => api<Driver>(`/drivers/${id}`),
  create: (name: string) => api<Driver>('/drivers', { method: 'POST', body: JSON.stringify({ name }) }),
  rename: (id: number, name: string) =>
    api<Driver>(`/drivers/${id}`, { method: 'PATCH', body: JSON.stringify({ name }) }),
  remove: (id: number) => api<Driver>(`/drivers/${id}`, { method: 'DELETE' }),
}

export const driverKeys = {
  all: ['drivers'] as const,
  page: (filter: DriverFilter, offset: number) => ['drivers', 'page', filter, offset] as const,
  index: ['drivers', 'index'] as const,
  one: (id: number) => ['drivers', 'one', id] as const,
}

export function useDriverPage(filter: DriverFilter, offset: number) {
  return useQuery({
    queryKey: driverKeys.page(filter, offset),
    queryFn: async () => toPage(await driversApi.list({ ...filter, ...pageParams(offset) })),
    placeholderData: keepPreviousData,
  })
}

export function useDriverIndex() {
  return useQuery({
    queryKey: driverKeys.index,
    queryFn: () => driversApi.list({ limit: LOOKUP_LIMIT }),
    select: indexById,
  })
}

export function useDriver(id: number) {
  return useQuery({ queryKey: driverKeys.one(id), queryFn: () => driversApi.get(id) })
}

export const useCreateDriver = () => useApiMutation(driversApi.create, [driverKeys.all])

export const useRenameDriver = () =>
  useApiMutation(({ id, name }: { id: number; name: string }) => driversApi.rename(id, name), [driverKeys.all])

export const useDeleteDriver = () => useApiMutation(driversApi.remove, [driverKeys.all])
