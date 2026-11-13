import { lazy, Suspense } from 'react'
import { LOOKUP_LIMIT } from '../../api/query'
import { formatDecimal, formatDuration, formatInt } from '../../lib/format'
import type { TelemetryEvent } from '../../types'
import { useRules } from '../alerts/api'
import { SpeedChart } from './SpeedChart'

// maplibre тянется только на страницы с треком, а не в каждую таблицу
const TrackMap = lazy(() => import('../../map/TrackMap').then((m) => ({ default: m.TrackMap })))

// useSpeedLimit — порог включённого правила превышения скорости: его рисуем на графике
export function useSpeedLimit(): number | undefined {
  const rules = useRules()
  return rules.data?.find((r) => r.type === 'SPEED_EXCEEDED' && r.enabled)?.threshold
}

// TrackView — трек на карте, сводка и график скорости по точкам в хронологическом порядке
export function TrackView({ points, fitKey }: { points: TelemetryEvent[]; fitKey: string }) {
  const speedLimit = useSpeedLimit()
  const first = points[0]
  const last = points[points.length - 1]
  const peak = Math.max(0, ...points.map((p) => p.speed_kmh))
  const over = speedLimit === undefined ? 0 : points.filter((p) => p.speed_kmh > speedLimit).length

  return (
    <div className="track">
      <Suspense fallback={<div className="track-map" />}>
        <TrackMap points={points} fitKey={fitKey} />
      </Suspense>
      <dl className="stats stats-row">
        <div>
          <dt>Точек</dt>
          <dd>{formatInt(points.length)}</dd>
        </div>
        <div>
          <dt>За время</dt>
          <dd>{first && last ? formatDuration(Date.parse(last.received_at) - Date.parse(first.received_at)) : '—'}</dd>
        </div>
        <div>
          <dt>Максимальная</dt>
          <dd data-over={(speedLimit !== undefined && peak > speedLimit) || undefined}>{Math.round(peak)} км/ч</dd>
        </div>
        <div>
          <dt>Точек сверх порога</dt>
          <dd data-over={over > 0 || undefined}>{speedLimit === undefined ? '—' : formatInt(over)}</dd>
        </div>
        <div>
          <dt>Топливо сейчас</dt>
          <dd>{last?.fuel != null ? `${formatDecimal(last.fuel)}%` : '—'}</dd>
        </div>
      </dl>
      {points.length >= LOOKUP_LIMIT && (
        <p className="field-note">
          Показаны последние {formatInt(LOOKUP_LIMIT)} точек — больше бэк за один запрос не отдаёт. Чтобы увидеть
          начало, выберите период короче.
        </p>
      )}
      <SpeedChart points={points} threshold={speedLimit} />
    </div>
  )
}
