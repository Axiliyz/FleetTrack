import { useState } from 'react'
import { formatTime } from '../lib/format'
import { acknowledgeAlert } from '../state/connect'
import { useFleet } from '../state/fleet'
import { Plate } from './Plate'

const COLLAPSED = 4
const EXPANDED = 40

export function EventLog() {
  const [expanded, setExpanded] = useState(false)
  const log = useFleet((s) => s.log)
  const vehicles = useFleet((s) => s.vehicles)
  const alerts = useFleet((s) => s.alerts)
  const select = useFleet((s) => s.select)

  const entries = log.slice(0, expanded ? EXPANDED : COLLAPSED)

  return (
    <section className="log" data-expanded={expanded || undefined} aria-label="Журнал событий">
      <header className="log-head">
        <h2>Журнал</h2>
        <button type="button" className="btn btn-quiet" aria-expanded={expanded} onClick={() => setExpanded(!expanded)}>
          {expanded ? 'Свернуть' : 'Развернуть'}
        </button>
      </header>
      {entries.length === 0 ? (
        <p className="log-empty">Пока тихо. Тревоги и рейсы появятся здесь, как только случатся.</p>
      ) : (
        <ol className="log-list">
          {entries.map((e) => {
            const vehicle = vehicles[e.vehicleId]
            const alert = e.alertId !== undefined ? alerts[e.alertId] : undefined
            return (
              <li key={e.key} data-tone={e.tone}>
                <time dateTime={e.at}>{formatTime(e.at)}</time>
                <span className="lamp" aria-hidden />
                <span className="log-text">{e.text}</span>
                {vehicle && (
                  <button
                    type="button"
                    className="log-plate"
                    aria-label={`Показать ${vehicle.number_plate}`}
                    onClick={() => select(vehicle.id, { fly: true })}
                  >
                    <Plate value={vehicle.number_plate} size="sm" />
                  </button>
                )}
                {alert?.status === 'FIRED' && (
                  <button type="button" className="btn btn-alarm btn-sm" onClick={() => void acknowledgeAlert(alert.id)}>
                    Принять
                  </button>
                )}
              </li>
            )
          })}
        </ol>
      )}
    </section>
  )
}
