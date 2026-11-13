import { useState, type ReactNode } from 'react'

// Панель фильтров над таблицей: все фильтры в один ряд, «Сбросить» — когда есть что сбрасывать
export function FilterBar({ active, onReset, children }: { active: boolean; onReset: () => void; children: ReactNode }) {
  return (
    <div className="filters" role="search">
      {children}
      {active && (
        <button type="button" className="btn btn-sm btn-quiet" onClick={onReset}>
          Сбросить
        </button>
      )}
    </div>
  )
}

export function FilterSelect({
  label,
  value,
  onChange,
  options,
  anyLabel = 'Все',
}: {
  label: string
  value: string
  onChange: (value: string) => void
  options: Array<{ value: string; label: string }>
  anyLabel?: string
}) {
  return (
    <label className="filter">
      <span>{label}</span>
      <select value={value} onChange={(e) => onChange(e.target.value)}>
        <option value="">{anyLabel}</option>
        {options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </label>
  )
}

// FilterText применяет значение по Enter или при уходе с поля, а не на каждую букву:
// фильтры бэка точные, и запрос на полуслове ничего не найдёт
export function FilterText({
  label,
  value,
  onCommit,
  placeholder,
  normalize = (v) => v.trim(),
}: {
  label: string
  value: string
  onCommit: (value: string) => void
  placeholder?: string
  normalize?: (value: string) => string
}) {
  const [draft, setDraft] = useState(value)
  const [synced, setSynced] = useState(value)
  // фильтр сбросили снаружи — поле тоже очищается
  if (value !== synced) {
    setSynced(value)
    setDraft(value)
  }

  const commit = () => {
    const next = normalize(draft)
    setDraft(next)
    if (next !== value) onCommit(next)
  }

  return (
    <label className="filter">
      <span>{label}</span>
      <input
        type="search"
        value={draft}
        placeholder={placeholder}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={commit}
        onKeyDown={(e) => {
          if (e.key === 'Enter') commit()
        }}
      />
    </label>
  )
}

export function FilterDate({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <label className="filter">
      <span>{label}</span>
      <input type="date" value={value} onChange={(e) => onChange(e.target.value)} />
    </label>
  )
}
