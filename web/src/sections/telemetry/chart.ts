// Геометрия графика скорости: чистые функции без DOM, их удобно проверять тестами
import type { TelemetryEvent } from '../../types'

export const CHART_HEIGHT = 180
export const CHART_PAD = { top: 12, right: 12, bottom: 24, left: 36 }
// разрыв в данных длиннее этого — трекер молчал: линию не тянем через дыру
export const GAP_MS = 5 * 60_000
export const Y_STEP = 20

export interface Plot {
  x: (t: number) => number
  y: (v: number) => number
  yMax: number
  times: number[]
}

// buildPlot — шкалы по времени и скорости. Верх шкалы — с запасом над пиком и над порогом,
// чтобы пунктир порога всегда был виден
export function buildPlot(points: TelemetryEvent[], width: number, threshold: number | undefined): Plot {
  const times = points.map((p) => Date.parse(p.received_at))
  const t0 = times[0]
  const t1 = Math.max(times[times.length - 1], t0 + 1)
  const peak = Math.max(...points.map((p) => p.speed_kmh), threshold ?? 0, Y_STEP)
  const yMax = Math.ceil((peak * 1.1) / Y_STEP) * Y_STEP
  const innerW = width - CHART_PAD.left - CHART_PAD.right
  const innerH = CHART_HEIGHT - CHART_PAD.top - CHART_PAD.bottom
  return {
    x: (t) => CHART_PAD.left + ((t - t0) / (t1 - t0)) * innerW,
    y: (v) => CHART_PAD.top + innerH - (v / yMax) * innerH,
    yMax,
    times,
  }
}

// linePath — путь с разрывами там, где трекер молчал дольше GAP_MS
export function linePath(points: TelemetryEvent[], plot: Plot): string {
  return points
    .map((p, i) => {
      const gap = i > 0 && plot.times[i] - plot.times[i - 1] > GAP_MS
      const cmd = i === 0 || gap ? 'M' : 'L'
      return `${cmd}${plot.x(plot.times[i]).toFixed(1)},${plot.y(p.speed_kmh).toFixed(1)}`
    })
    .join('')
}

// nearest — индекс точки, ближайшей по времени к t; times отсортированы по возрастанию
export function nearest(times: number[], t: number): number {
  let lo = 0
  let hi = times.length - 1
  while (lo < hi) {
    const mid = (lo + hi) >> 1
    if (times[mid] < t) lo = mid + 1
    else hi = mid
  }
  return lo > 0 && t - times[lo - 1] < times[lo] - t ? lo - 1 : lo
}
