// Ответы в формате API: {status, data} при успехе, {status, message} при ошибке
import { vi } from 'vitest'

export const json = (data: unknown, status = 200) =>
  new Response(JSON.stringify(status < 400 ? { status: 'success', data } : { status: 'error', message: data }), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })

export interface Call {
  method: string
  url: string
  body: unknown
}

// recordFetch подменяет fetch: запоминает запросы и отвечает тем, что вернёт respond
export function recordFetch(respond: (call: Call) => Response = () => json(null)) {
  const calls: Call[] = []
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const call = {
        method: init?.method ?? 'GET',
        url: String(input),
        body: init?.body ? JSON.parse(String(init.body)) : undefined,
      }
      calls.push(call)
      return respond(call)
    }),
  )
  return calls
}
