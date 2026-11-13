import { useEffect, useRef, useState, type PointerEvent } from 'react'
import { formatTime, formatTimeSec } from '../../lib/format'
import type { TelemetryEvent } from '../../types'
import { CHART_HEIGHT as HEIGHT, CHART_PAD as PAD, Y_STEP, buildPlot, linePath, nearest } from './chart'

// SpeedChart — скорость во времени: одна линия, порог правила пунктиром, перекрестие с подсказкой
export function SpeedChart({ points, threshold }: { points: TelemetryEvent[]; threshold?: number }) {
  const wrapRef = useRef<HTMLDivElement>(null)
  const [width, setWidth] = useState(0)
  const [hover, setHover] = useState<number | null>(null)

  useEffect(() => {
    const el = wrapRef.current
    if (!el) return
    const observer = new ResizeObserver(([entry]) => setWidth(Math.round(entry.contentRect.width)))
    observer.observe(el)
    return () => observer.disconnect()
  }, [])

  // обёртка рендерится всегда: за ней следит ResizeObserver, даже пока точек мало
  const plot = width > 0 && points.length >= 2 ? buildPlot(points, width, threshold) : null
  const ticks = plot ? Array.from({ length: plot.yMax / Y_STEP + 1 }, (_, i) => i * Y_STEP) : []
  const hovered = hover !== null ? points[hover] : null

  const onMove = (e: PointerEvent<SVGSVGElement>) => {
    if (!plot) return
    const rect = e.currentTarget.getBoundingClientRect()
    const x = e.clientX - rect.left
    const t0 = plot.times[0]
    const t1 = plot.times[plot.times.length - 1]
    const ratio = (x - PAD.left) / (width - PAD.left - PAD.right)
    setHover(nearest(plot.times, t0 + Math.min(Math.max(ratio, 0), 1) * (t1 - t0)))
  }

  return (
    <figure className="chart">
      <figcaption className="chart-title">Скорость, км/ч</figcaption>
      <div ref={wrapRef} className="chart-plot">
        {points.length < 2 && <p className="chart-empty">Для графика нужно хотя бы две точки.</p>}
        {plot && (
          <svg
            width={width}
            height={HEIGHT}
            role="img"
            aria-label={`График скорости: ${points.length} точек с ${formatTime(plot.times[0])} до ${formatTime(plot.times[plot.times.length - 1])}`}
            onPointerMove={onMove}
            onPointerLeave={() => setHover(null)}
          >
            {ticks.map((v) => (
              <g key={v} className="chart-grid">
                <line x1={PAD.left} x2={width - PAD.right} y1={plot.y(v)} y2={plot.y(v)} />
                <text x={PAD.left - 6} y={plot.y(v)} dy="0.32em" textAnchor="end">
                  {v}
                </text>
              </g>
            ))}
            <text className="chart-axis" x={PAD.left} y={HEIGHT - 6}>
              {formatTime(plot.times[0])}
            </text>
            <text className="chart-axis" x={width - PAD.right} y={HEIGHT - 6} textAnchor="end">
              {formatTime(plot.times[plot.times.length - 1])}
            </text>

            {threshold !== undefined && (
              <g className="chart-threshold">
                <line x1={PAD.left} x2={width - PAD.right} y1={plot.y(threshold)} y2={plot.y(threshold)} />
                <text x={width - PAD.right} y={plot.y(threshold)} dy="-0.4em" textAnchor="end">
                  порог {threshold}
                </text>
              </g>
            )}

            <path className="chart-line" d={linePath(points, plot)} />

            {hovered && hover !== null && (
              <g className="chart-cursor">
                <line x1={plot.x(plot.times[hover])} x2={plot.x(plot.times[hover])} y1={PAD.top} y2={HEIGHT - PAD.bottom} />
                <circle cx={plot.x(plot.times[hover])} cy={plot.y(hovered.speed_kmh)} r={4} />
              </g>
            )}
          </svg>
        )}
        {plot && hovered && hover !== null && (
          <div
            className="chart-tip"
            style={{ left: plot.x(plot.times[hover]), top: plot.y(hovered.speed_kmh) }}
            data-flip={plot.x(plot.times[hover]) > width / 2 || undefined}
          >
            <b>{Math.round(hovered.speed_kmh)} км/ч</b>
            <span>{formatTimeSec(hovered.received_at)}</span>
            {hovered.fuel !== null && <span>топливо {Math.round(hovered.fuel)}%</span>}
          </div>
        )}
      </div>
    </figure>
  )
}
