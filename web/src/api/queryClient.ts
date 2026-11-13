import { QueryClient } from '@tanstack/react-query'
import { LIVE } from '../config'
import { ApiError, auth } from './client'

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // у симулятора бэкенда нет: разделы показывают подсказку, запросы не уходят
      enabled: LIVE,
      staleTime: 15_000,
      // 4xx не исправится повтором: нет прав, нет записи, кривой фильтр
      retry: (failures, err) => !(err instanceof ApiError && err.status < 500) && failures < 2,
    },
  },
})

// данные прошлого пользователя не должны мелькнуть следующему
auth.onChange((session) => {
  if (!session) queryClient.clear()
})
