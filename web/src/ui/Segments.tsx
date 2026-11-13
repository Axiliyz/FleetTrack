// Segments — сегментная шкала, как на приборной панели.
// Зона за порогом слегка подкрашена всегда, а горит красным, только когда значение действительно за порогом:
// выше порога для скорости, ниже — для топлива.
export function Segments({
  value,
  max,
  count,
  threshold,
  danger,
  label,
}: {
  value: number
  max: number
  count: number
  threshold?: number
  danger: 'above' | 'below'
  label: string
}) {
  const lit = Math.round((Math.min(Math.max(value, 0), max) / max) * count)
  const step = max / count
  const alarm = threshold !== undefined && (danger === 'above' ? value > threshold : value < threshold)
  return (
    <span
      className="segments"
      role="meter"
      aria-label={label}
      aria-valuemin={0}
      aria-valuemax={max}
      aria-valuenow={Math.round(value)}
    >
      {Array.from({ length: count }, (_, i) => {
        const from = i * step
        const zone =
          threshold !== undefined && (danger === 'above' ? from >= threshold : from + step <= threshold)
        const on = i < lit
        // скорость: красным горит только часть сверх порога; топливо: вся шкала, когда бак ниже порога
        const hot = on && (danger === 'above' ? zone : alarm)
        return <i key={i} data-on={on || undefined} data-zone={zone || undefined} data-hot={hot || undefined} />
      })}
    </span>
  )
}
