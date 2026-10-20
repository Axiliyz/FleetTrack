import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { api, auth } from '../../api/client'
import type { FleetEvent, FleetSnapshot } from '../../types'
import { ApiFleetSource, ackErrorText } from './ApiFleetSource'
import { ApiError } from '../../api/client'

const SNAPSHOT: FleetSnapshot = {
  org: { id: 5, name: 'Северный тракт' },
  vehicles: [{ id: 3, organization_id: 5, vin: 'XTA21099012345678', number_plate: 'A123BC77', model: 'КАМАЗ', status: 'ON_TRIP' }],
  drivers: [],
  trips: [],
  positions: [],
  alerts: [],
  rules: [],
}

const json = (data: unknown, status = 200) =>
  new Response(JSON.stringify(status < 400 ? { status: 'success', data } : { status: 'error', message: data }), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })

// sseStream — управляемый поток: тест сам решает, когда отправить событие
function sseStream() {
  const encoder = new TextEncoder()
  let push: (chunk: string) => void = () => {}
  const body = new ReadableStream<Uint8Array>({
    start(controller) {
      push = (chunk) => controller.enqueue(encoder.encode(chunk))
    },
  })
  return {
    response: new Response(body, { status: 200, headers: { 'Content-Type': 'text/event-stream' } }),
    send: (type: string, data: unknown) => push(`event: ${type}\ndata: ${JSON.stringify(data)}\n\n`),
  }
}

const tick = (ms = 0) => new Promise((r) => setTimeout(r, ms))

async function waitUntil(cond: () => boolean) {
  for (let i = 0; i < 200 && !cond(); i++) await tick(5)
  expect(cond()).toBe(true)
}

beforeEach(async () => {
  vi.stubGlobal('window', globalThis)
  // библиотека при закрытии потока всегда снимает слушатель visibilitychange с document
  vi.stubGlobal('document', { hidden: false, addEventListener: () => {}, removeEventListener: () => {} })
  vi.stubGlobal(
    'fetch',
    vi.fn(async () => json({ access_token: 'access-1', refresh_token: 'refresh-1' })),
  )
  await auth.login('dispatcher@fleettrack.test', 'irrelevant')
})

afterEach(async () => {
  vi.stubGlobal('fetch', vi.fn(async () => json(null)))
  await auth.logout()
  vi.unstubAllGlobals()
})

describe('ApiFleetSource', () => {
  it('открывает поток раньше снимка и придерживает события, пока снимок грузится', async () => {
    const stream = sseStream()
    let releaseSnapshot: () => void = () => {}
    const snapshotGate = new Promise<void>((r) => (releaseSnapshot = r))
    const calls: string[] = []

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input)
        calls.push(url)
        if (url.endsWith('/stream')) return stream.response
        if (url.endsWith('/fleet/snapshot')) {
          await snapshotGate
          return json(SNAPSHOT)
        }
        throw new Error(`unexpected ${url}`)
      }),
    )

    const order: string[] = []
    const unsubscribe = new ApiFleetSource().subscribe({
      onEvent: (e: FleetEvent) => order.push(`${e.type}:${'id' in e.data ? e.data.id : ''}`),
      onSnapshot: () => order.push('snapshot'),
    })

    await waitUntil(() => calls.some((u) => u.endsWith('/fleet/snapshot')))
    expect(calls[0]).toMatch(/\/stream$/)

    stream.send('alert', { id: 7, status: 'FIRED' })
    await tick(20)
    expect(order).toEqual([]) // событие держится в буфере, пока нет снимка

    releaseSnapshot()
    await waitUntil(() => order.length === 2)
    expect(order).toEqual(['snapshot', 'alert:7'])

    stream.send('vehicle', { id: 3, status: 'IDLE' })
    await waitUntil(() => order.length === 3)
    expect(order[2]).toBe('vehicle:3')
    unsubscribe()
  })

  it('на 401 обновляет токен и переподключается с новым', async () => {
    const stream = sseStream()
    const streamAuth: string[] = []
    let refreshCalls = 0

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input)
        if (url.endsWith('/stream')) {
          const header = new Headers(init?.headers).get('Authorization') ?? ''
          streamAuth.push(header)
          return header === 'Bearer access-2' ? stream.response : json('expired', 401)
        }
        if (url.endsWith('/refresh')) {
          refreshCalls++
          return json({ access_token: 'access-2', refresh_token: 'refresh-2' })
        }
        if (url.endsWith('/fleet/snapshot')) return json(SNAPSHOT)
        throw new Error(`unexpected ${url}`)
      }),
    )

    let snapshots = 0
    const unsubscribe = new ApiFleetSource().subscribe({ onEvent: () => {}, onSnapshot: () => snapshots++ })
    await waitUntil(() => snapshots === 1)

    expect(streamAuth).toEqual(['Bearer access-1', 'Bearer access-2'])
    expect(refreshCalls).toBe(1)
    expect(auth.current()?.refreshToken).toBe('refresh-2')
    unsubscribe()
  })

  it('принимает алерт через PATCH /alerts/{id}', async () => {
    const fetchMock = vi.fn(async () => json({ id: 7, status: 'ACKNOWLEDGED' }))
    vi.stubGlobal('fetch', fetchMock)

    await new ApiFleetSource().acknowledgeAlert(7)

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toBe('/api/alerts/7')
    expect(init.method).toBe('PATCH')
    expect(JSON.parse(String(init.body))).toEqual({ status: 'ACKNOWLEDGED' })
  })

  it('переводит ответы на «Принять» в понятный текст', () => {
    expect(ackErrorText(new ApiError(409, 'alert is already acknowledged or resolved'))).toBe('алерт уже принят или закрыт')
    expect(ackErrorText(new ApiError(403, 'forbidden'))).toBe('недостаточно прав')
    expect(ackErrorText(new TypeError('Failed to fetch'))).toBe('нет связи с сервером')
  })
})

describe('api client', () => {
  it('одновременные 401 обновляют токен один раз — refresh-токен одноразовый', async () => {
    let refreshCalls = 0
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input)
        if (url.endsWith('/refresh')) {
          refreshCalls++
          await tick(10)
          return json({ access_token: 'access-2', refresh_token: 'refresh-2' })
        }
        const header = new Headers(init?.headers).get('Authorization')
        return header === 'Bearer access-2' ? json({ ok: true }) : json('expired', 401)
      }),
    )

    const results = await Promise.all([api('/alerts'), api('/fleet/snapshot'), api('/alerts?status=FIRED')])
    expect(results).toEqual([{ ok: true }, { ok: true }, { ok: true }])
    expect(refreshCalls).toBe(1)
  })

  it('если refresh не удался, сессия завершается', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => (String(input).endsWith('/refresh') ? json('revoked', 401) : json('expired', 401))),
    )
    await expect(api('/alerts')).rejects.toMatchObject({ status: 401 })
    expect(auth.current()).toBeNull()
  })
})
