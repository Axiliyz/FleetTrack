import { splitPlate } from '../lib/format'

// Plate — номер в виде настоящего российского знака: основная часть и регион с RUS
export function Plate({ value, size = 'md' }: { value: string; size?: 'sm' | 'md' | 'lg' }) {
  const parts = splitPlate(value)
  if (!parts) {
    return (
      <span className="plate" data-size={size}>
        <span className="plate-main">{value}</span>
      </span>
    )
  }
  return (
    <span className="plate" data-size={size} translate="no">
      <span className="plate-main">{parts.main}</span>
      <span className="plate-region">
        <span>{parts.region}</span>
        <small>RUS</small>
      </span>
    </span>
  )
}
