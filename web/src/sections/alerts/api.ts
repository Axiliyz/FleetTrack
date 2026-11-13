import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { api } from '../../api/client'
import { useApiMutation } from '../../api/mutations'
import { pageParams, toPage, apiList, type QueryParams } from '../../api/query'
import { fromInputDate } from '../../lib/format'
import type { Alert, AlertRule, AlertType, Severity } from '../../types'

export const ALERT_FILTER_KEYS = ['status', 'severity', 'vehicle_id', 'created_from'] as const
export type AlertFilter = Record<(typeof ALERT_FILTER_KEYS)[number], string>

// NewRule — dto.CreateAlertRuleRequest без organization_id: бэк берёт организацию из токена
export interface NewRule {
  type: AlertType
  name: string
  threshold: number
  severity: Severity
  enabled: boolean
}

export const alertsApi = {
  list: (params: QueryParams) => apiList<Alert>('/alerts', params),
  acknowledge: (id: number) =>
    api<Alert>(`/alerts/${id}`, { method: 'PATCH', body: JSON.stringify({ status: 'ACKNOWLEDGED' }) }),
  rules: () => apiList<AlertRule>('/alert-rules'),
  createRule: (rule: NewRule) => api<AlertRule>('/alert-rules', { method: 'POST', body: JSON.stringify(rule) }),
  removeRule: (id: number) => api<AlertRule>(`/alert-rules/${id}`, { method: 'DELETE' }),
}

export const alertKeys = {
  all: ['alerts'] as const,
  page: (filter: AlertFilter, offset: number) => ['alerts', 'page', filter, offset] as const,
  rules: ['alert-rules'] as const,
}

export function useAlertPage(filter: AlertFilter, offset: number) {
  return useQuery({
    queryKey: alertKeys.page(filter, offset),
    queryFn: async () =>
      toPage(
        await alertsApi.list({ ...filter, created_from: fromInputDate(filter.created_from), ...pageParams(offset) }),
      ),
    placeholderData: keepPreviousData,
    // журнал пополняется сам по себе: движок правил работает на бэке
    refetchInterval: 30_000,
  })
}

export function useRules() {
  return useQuery({ queryKey: alertKeys.rules, queryFn: alertsApi.rules })
}

export const useAcknowledge = () => useApiMutation(alertsApi.acknowledge, [alertKeys.all])
export const useCreateRule = () => useApiMutation(alertsApi.createRule, [alertKeys.rules])
export const useDeleteRule = () => useApiMutation(alertsApi.removeRule, [alertKeys.rules])
