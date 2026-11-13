import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { api } from '../../api/client'
import { useApiMutation } from '../../api/mutations'
import { pageParams, toPage, apiList, type QueryParams } from '../../api/query'
import type { Org, User, UserRole } from '../../types'

export const USER_FILTER_KEYS = ['role'] as const
export type UserFilter = Record<(typeof USER_FILTER_KEYS)[number], string>

// NewUser — dto.CreateUserRequest без organization_id: пользователь всегда заводится в своей организации
export interface NewUser {
  name: string
  email: string
  password: string
  role: UserRole
  // обязателен для роли DRIVER: водитель видит только свои рейсы
  driver_id?: number
}

export const teamApi = {
  list: (params: QueryParams) => apiList<User>('/users', params),
  create: (user: NewUser) => api<User>('/users', { method: 'POST', body: JSON.stringify(user) }),
  remove: (id: number) => api<User>(`/users/${id}`, { method: 'DELETE' }),
  // каждый видит только свою организацию — список из одного элемента
  organization: async () => (await apiList<Org>('/organizations'))[0] ?? null,
}

export const teamKeys = {
  all: ['users'] as const,
  page: (filter: UserFilter, offset: number) => ['users', 'page', filter, offset] as const,
  org: ['organization'] as const,
}

export function useUserPage(filter: UserFilter, offset: number) {
  return useQuery({
    queryKey: teamKeys.page(filter, offset),
    queryFn: async () => toPage(await teamApi.list({ ...filter, ...pageParams(offset) })),
    placeholderData: keepPreviousData,
  })
}

export function useOrganization() {
  return useQuery({ queryKey: teamKeys.org, queryFn: teamApi.organization, staleTime: Infinity })
}

export const useCreateUser = () => useApiMutation(teamApi.create, [teamKeys.all])
export const useDeleteUser = () => useApiMutation(teamApi.remove, [teamKeys.all])
