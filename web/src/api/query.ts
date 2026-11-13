import { api } from './client'

// Строка запроса для списков API: пустые значения не отправляем — бэк трактует пустой параметр
// как «фильтр не задан», но лишние ключи только засоряют URL и ключи кеша
export type QueryParams = Record<string, string | number | null | undefined>

// apiList — список с API. Часть хендлеров на пустой выборке отдаёт data: null вместо [] (nil-слайс в Go),
// поэтому приводим к массиву здесь, а не проверяем в каждой таблице
export async function apiList<T>(path: string, params: QueryParams = {}): Promise<T[]> {
  return (await api<T[] | null>(withQuery(path, params))) ?? []
}

export function withQuery(path: string, params: QueryParams): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value === null || value === undefined || value === '') continue
    search.set(key, String(value))
  }
  const qs = search.toString()
  return qs ? `${path}?${qs}` : path
}

// Списки бэка отдают голый массив без total. Чтобы понять, есть ли следующая страница,
// просим на одну запись больше, чем показываем
export const PAGE_SIZE = 50

export interface Page<T> {
  rows: T[]
  hasNext: boolean
}

export const pageParams = (offset: number) => ({ limit: PAGE_SIZE + 1, offset })

export function toPage<T>(rows: T[]): Page<T> {
  return { rows: rows.slice(0, PAGE_SIZE), hasNext: rows.length > PAGE_SIZE }
}

// LOOKUP_LIMIT — потолок limit на бэке; справочники для подписей и выпадающих списков берём целиком
export const LOOKUP_LIMIT = 500
