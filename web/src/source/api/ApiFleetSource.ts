// Живые данные: снимок через GET /fleet/snapshot и события через SSE GET /stream.
// Обычный EventSource не умеет передавать заголовок Authorization, поэтому поток читается fetch-ем
import { fetchEventSource, type EventSourceMessage } from '@microsoft/fetch-event-source'
import { API_BASE, ApiError, api, auth } from '../../api/client'
import type { FleetEvent, FleetSnapshot } from '../../types'
import type { FleetSource, SubscribeHandlers } from '../FleetSource'

const EVENT_TYPES = new Set<FleetEvent['type']>(['telemetry', 'alert', 'trip', 'vehicle'])
const RETRY_MS = 3000

// RetryLater — обрыв, после которого надо переподключиться (сеть, рестарт сервера, нас отключили как медленных)
class RetryLater extends Error {}
// RetryNow — токен только что обновлён, ждать незачем
class RetryNow extends Error {}
// StopStream — переподключаться бессмысленно (нет прав, сессия кончилась)
class StopStream extends Error {}

export class ApiFleetSource implements FleetSource {
  readonly snapshotOnConnect = true

  load(): Promise<FleetSnapshot> {
    return api<FleetSnapshot>('/fleet/snapshot')
  }

  subscribe(handlers: SubscribeHandlers): () => void {
    const controller = new AbortController()
    // пока грузится снимок, события копятся здесь и применяются после него
    let buffer: FleetEvent[] | null = null

    const resync = async () => {
      buffer = []
      try {
        const snapshot = await this.load()
        if (controller.signal.aborted) return
        handlers.onSnapshot?.(snapshot)
        for (const event of buffer) handlers.onEvent(event)
      } catch (err) {
        if (!controller.signal.aborted) console.error('[fleet] snapshot failed', err)
      } finally {
        buffer = null
      }
    }

    handlers.onLinkState?.('connecting')
    void fetchEventSource(`${API_BASE}/stream`, {
      signal: controller.signal,
      // без этого библиотека рвёт поток, когда вкладка скрыта, и каждый возврат стоил бы нового снимка
      openWhenHidden: true,
      // токен берём заново на каждую попытку: после refresh старый уже недействителен
      fetch: (input, init) => fetch(input, { ...init, headers: { ...init?.headers, ...auth.authHeader() } }),

      async onopen(res) {
        if (res.ok && res.headers.get('content-type')?.startsWith('text/event-stream')) {
          handlers.onLinkState?.('live')
          // снимок — уже после открытия потока: всё, что случится дальше, придёт событиями
          void resync()
          return
        }
        if (res.status === 401) {
          if (await auth.refresh()) throw new RetryNow()
          throw new StopStream('сессия истекла')
        }
        if (res.status >= 400 && res.status < 500) throw new StopStream(`поток отклонён: HTTP ${res.status}`)
        throw new RetryLater()
      },

      onmessage(msg: EventSourceMessage) {
        const type = msg.event as FleetEvent['type']
        if (!EVENT_TYPES.has(type) || !msg.data) return
        let event: FleetEvent
        try {
          event = { type, data: JSON.parse(msg.data) } as FleetEvent
        } catch {
          console.warn('[fleet] bad event payload', msg.data)
          return
        }
        if (buffer) buffer.push(event)
        else handlers.onEvent(event)
      },

      onclose() {
        // сервер закрыл поток сам: перезапуск или мы не успевали читать — переподключаемся
        handlers.onLinkState?.('reconnecting')
        throw new RetryLater()
      },

      onerror(err) {
        if (err instanceof StopStream) {
          console.error('[fleet]', err.message)
          throw err
        }
        if (err instanceof RetryNow) return 0
        handlers.onLinkState?.('reconnecting')
        return RETRY_MS
      },
    }).catch((err: unknown) => {
      if (!controller.signal.aborted) console.error('[fleet] stream stopped', err)
    })

    return () => controller.abort()
  }

  async acknowledgeAlert(alertId: number): Promise<void> {
    // ответ не применяем: изменение придёт событием из потока, как у всех диспетчеров
    await api(`/alerts/${alertId}`, { method: 'PATCH', body: JSON.stringify({ status: 'ACKNOWLEDGED' }) })
  }
}

// ackErrorText переводит ответ API на «Принять» в понятную диспетчеру фразу
export function ackErrorText(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.status === 409) return 'алерт уже принят или закрыт'
    if (err.status === 403) return 'недостаточно прав'
    if (err.status === 404) return 'алерт не найден'
    return err.message
  }
  return 'нет связи с сервером'
}
