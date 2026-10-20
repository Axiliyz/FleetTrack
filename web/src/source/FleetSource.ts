import type { FleetEvent, FleetSnapshot } from '../types'

export type LinkState = 'connecting' | 'live' | 'reconnecting'

// SubscribeHandlers — куда источник отдаёт данные
export interface SubscribeHandlers {
  onEvent(event: FleetEvent): void
  // свежий снимок: источник с живым потоком присылает его при каждом (пере)подключении,
  // чтобы карта не жила с дырой из событий, пропущенных во время обрыва
  onSnapshot?(snapshot: FleetSnapshot): void
  onLinkState?(state: LinkState): void
}

// FleetSource — откуда фронт берёт данные: симулятор или REST + SSE с тем же интерфейсом
export interface FleetSource {
  // true — снимок приходит через onSnapshot после подключения, load() вызывать не нужно
  readonly snapshotOnConnect?: boolean
  load(): Promise<FleetSnapshot>
  // subscribe возвращает функцию отписки
  subscribe(handlers: SubscribeHandlers): () => void
  acknowledgeAlert(alertId: number): Promise<void>
  // управление временем есть только у симулятора
  sim?: SimControls
}

export interface SimControls {
  now(): number
  scale(): number
  // 0 — пауза
  setScale(scale: number): void
}
