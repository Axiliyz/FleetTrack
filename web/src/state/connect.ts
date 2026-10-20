// connect связывает источник данных со стором и анимацией
import type { FleetSource } from '../source/FleetSource'
import { ackErrorText } from '../source/api/ApiFleetSource'
import type { FleetEvent, FleetSnapshot } from '../types'
import { motion } from '../map/motion'
import { playAlarm } from '../lib/sound'
import { useFleet } from './fleet'
import { severityRank } from './selectors'

let current: FleetSource | null = null

export function connect(source: FleetSource): () => void {
  current = source
  motion.reset()
  let unsubscribe = () => {}
  let cancelled = false
  const store = useFleet.getState()
  store.setLink(source.snapshotOnConnect ? 'connecting' : null)

  const applySnapshot = (snapshot: FleetSnapshot) => {
    if (cancelled) return
    const scale = source.sim?.scale() ?? null
    useFleet.getState().init(snapshot, scale)
    motion.setPaused(scale === 0)
    for (const p of snapshot.positions) motion.push(p.vehicle_id, [p.lon, p.lat])
  }

  const subscribe = () => {
    unsubscribe = source.subscribe({
      onEvent,
      onSnapshot: applySnapshot,
      onLinkState: (state) => useFleet.getState().setLink(state),
    })
  }

  if (source.snapshotOnConnect) {
    // живой источник сам пришлёт снимок, как только откроет поток
    subscribe()
  } else {
    source
      .load()
      .then((snapshot) => {
        applySnapshot(snapshot)
        if (!cancelled) subscribe()
      })
      .catch((err: unknown) => {
        if (!cancelled) useFleet.getState().fail(err instanceof Error ? err.message : String(err))
      })
  }

  return () => {
    cancelled = true
    unsubscribe()
    if (current === source) current = null
  }
}

function onEvent(event: FleetEvent): void {
  const store = useFleet.getState()
  if (store.status !== 'ready') return
  const isNewAlert = event.type === 'alert' && !store.alerts[event.data.id]
  store.apply(event)

  if (event.type === 'telemetry') {
    motion.push(event.data.vehicle_id, [event.data.lon, event.data.lat])
  }
  if (isNewAlert && event.type === 'alert' && event.data.status === 'FIRED' && severityRank(event.data.severity) >= 2) {
    store.showToast(event.data.id)
    if (!store.muted) playAlarm()
  }
}

export async function acknowledgeAlert(alertId: number): Promise<void> {
  if (!current) return
  try {
    await current.acknowledgeAlert(alertId)
  } catch (err) {
    const alert = useFleet.getState().alerts[alertId]
    useFleet.getState().note({
      tone: 'warn',
      text: `Не удалось принять: ${ackErrorText(err)}`,
      vehicleId: alert?.vehicle_id ?? 0,
    })
  }
}

export function setTimeScale(scale: number): void {
  if (!current?.sim) return
  current.sim.setScale(scale)
  motion.setPaused(scale === 0)
  useFleet.getState().setScale(scale)
}

export function togglePause(): void {
  const { timeScale, lastScale } = useFleet.getState()
  if (timeScale === null) return
  setTimeScale(timeScale === 0 ? lastScale : 0)
}
