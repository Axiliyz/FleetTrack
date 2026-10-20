import { useMemo } from 'react'
import { auth } from '../api/client'
import { LIVE } from '../config'
import { formatDate, formatTime } from '../lib/format'
import { setTimeScale } from '../state/connect'
import { useFleet } from '../state/fleet'
import { conditionsById, type Condition } from '../state/selectors'

const SCALES = [
  { value: 1, label: '×1', title: 'Реальное время' },
  { value: 60, label: '×60', title: 'Минута за секунду' },
  { value: 300, label: '×300', title: 'Пять минут за секунду' },
]

const COUNTERS: Array<{ cond: Condition; label: string }> = [
  { cond: 'trip', label: 'в рейсе' },
  { cond: 'idle', label: 'свободны' },
  { cond: 'service', label: 'на ТО' },
  { cond: 'alarm', label: 'тревога' },
  { cond: 'offline', label: 'без связи' },
]

export function TopBar() {
  const org = useFleet((s) => s.org)
  const vehicles = useFleet((s) => s.vehicles)
  const alerts = useFleet((s) => s.alerts)
  const clock = useFleet((s) => s.clock)
  const timeScale = useFleet((s) => s.timeScale)
  const muted = useFleet((s) => s.muted)
  const toggleMuted = useFleet((s) => s.toggleMuted)
  const rosterOpen = useFleet((s) => s.rosterOpen)
  const link = useFleet((s) => s.link)
  const setRosterOpen = useFleet((s) => s.setRosterOpen)

  const counts = useMemo(() => {
    const result: Record<Condition, number> = { trip: 0, idle: 0, service: 0, alarm: 0, offline: 0 }
    for (const c of conditionsById(vehicles, alerts).values()) result[c] += 1
    return result
  }, [vehicles, alerts])

  return (
    <header className="topbar">
      <button
        type="button"
        className="topbar-roster"
        aria-expanded={rosterOpen}
        aria-controls="roster"
        onClick={() => setRosterOpen(!rosterOpen)}
      >
        Парк
      </button>

      <div className="brand">
        <span className="brand-mark" aria-hidden />
        <span className="brand-name">FleetTrack</span>
        {org && <span className="brand-org">{org.name}</span>}
      </div>

      <ul className="counters" aria-label="Состояние парка">
        {COUNTERS.map(({ cond, label }) => (
          <li key={cond} data-cond={cond} data-zero={counts[cond] === 0 || undefined}>
            <span className="lamp" aria-hidden />
            <b>{counts[cond]}</b> {label}
          </li>
        ))}
      </ul>

      {link && link !== 'live' && (
        <p className="link-state" role="status" data-state={link}>
          <span className="lamp" aria-hidden />
          {link === 'connecting' ? 'Подключаюсь к потоку' : 'Нет связи с сервером, переподключаюсь'}
        </p>
      )}

      <div className="clock" aria-live="off">
        <span className="clock-time">{formatTime(clock)}</span>
        <span className="clock-date">{formatDate(clock)}</span>
      </div>

      {timeScale !== null && (
        <div className="timectl" role="group" aria-label="Скорость симуляции">
          <button
            type="button"
            aria-pressed={timeScale === 0}
            title="Пауза (P)"
            onClick={() => setTimeScale(timeScale === 0 ? useFleet.getState().lastScale : 0)}
          >
            {timeScale === 0 ? 'Пуск' : 'Пауза'}
          </button>
          {SCALES.map((s) => (
            <button
              key={s.value}
              type="button"
              aria-pressed={timeScale === s.value}
              title={s.title}
              onClick={() => setTimeScale(s.value)}
            >
              {s.label}
            </button>
          ))}
        </div>
      )}

      <button type="button" className="sound" aria-pressed={!muted} onClick={toggleMuted}>
        {muted ? 'Звук выкл.' : 'Звук вкл.'}
      </button>

      {LIVE && (
        <button type="button" className="sound" onClick={() => void auth.logout()}>
          Выйти
        </button>
      )}
    </header>
  )
}
