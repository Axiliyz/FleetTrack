const timeFmt = new Intl.DateTimeFormat('ru-RU', { hour: '2-digit', minute: '2-digit' })
const timeSecFmt = new Intl.DateTimeFormat('ru-RU', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
const dateFmt = new Intl.DateTimeFormat('ru-RU', { day: 'numeric', month: 'short' })
const dateTimeFmt = new Intl.DateTimeFormat('ru-RU', {
  day: 'numeric',
  month: 'short',
  hour: '2-digit',
  minute: '2-digit',
})
const intFmt = new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 0 })
const decimalFmt = new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 1 })

export const formatTime = (ms: number | string) => timeFmt.format(new Date(ms))
export const formatTimeSec = (ms: number | string) => timeSecFmt.format(new Date(ms))
export const formatDate = (ms: number) => dateFmt.format(new Date(ms))
export const formatDateTime = (ms: number | string) => dateTimeFmt.format(new Date(ms))
export const formatInt = (n: number) => intFmt.format(n)
export const formatDecimal = (n: number) => decimalFmt.format(n)

// fromInputDate переводит значение <input type="date"> (локальная дата) в RFC3339 для API.
// endOfDay: дата «по» включает весь день, а не только его полночь
export function fromInputDate(value: string, endOfDay = false): string | null {
  if (!value) return null
  const [y, m, d] = value.split('-').map(Number)
  const date = endOfDay ? new Date(y, m - 1, d, 23, 59, 59) : new Date(y, m - 1, d)
  return Number.isNaN(date.getTime()) ? null : date.toISOString()
}

// formatDuration: «3 ч 12 мин», «45 мин», «меньше минуты»
export function formatDuration(ms: number): string {
  const totalMin = Math.floor(ms / 60_000)
  if (totalMin < 1) return 'меньше минуты'
  const h = Math.floor(totalMin / 60)
  const m = totalMin % 60
  if (h === 0) return `${m} мин`
  return m === 0 ? `${h} ч` : `${h} ч ${m} мин`
}

// splitPlate разбирает номер «А123ВС77» на основную часть и регион
export function splitPlate(plate: string): { main: string; region: string } | null {
  const m = /^([А-ЯA-Z])(\d{3})([А-ЯA-Z]{2})(\d{2,3})$/u.exec(plate.replace(/\s/g, '').toUpperCase())
  return m ? { main: `${m[1]} ${m[2]} ${m[3]}`, region: m[4] } : null
}

export function initials(name: string): string {
  return name
    .split(/\s+/)
    .slice(0, 2)
    .map((part) => part[0] ?? '')
    .join('')
}
