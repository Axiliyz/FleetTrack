import type { ReactNode } from 'react'
import type { UseQueryResult } from '@tanstack/react-query'
import { errorText } from '../api/errors'
import { LIVE } from '../config'
import type { Tone } from '../state/fleet'

// Page — каркас раздела: заголовок с действиями, ниже содержимое с прокруткой
export function Page({
  title,
  lead,
  actions,
  needsApi = true,
  children,
}: {
  title: ReactNode
  lead?: ReactNode
  actions?: ReactNode
  needsApi?: boolean
  children: ReactNode
}) {
  if (needsApi && !LIVE) return <LiveOnly title={title} />
  return (
    <main className="page">
      <header className="page-head">
        <div className="page-title">
          <h1>{title}</h1>
          {lead && <p className="page-lead">{lead}</p>}
        </div>
        {actions && <div className="page-actions">{actions}</div>}
      </header>
      {children}
    </main>
  )
}

// LiveOnly — разделы работают с API, а у симулятора своего бэкенда нет
function LiveOnly({ title }: { title: ReactNode }) {
  return (
    <main className="page">
      <header className="page-head">
        <div className="page-title">
          <h1>{title}</h1>
        </div>
      </header>
      <p className="empty">
        Раздел работает с живым бэкендом. Запустите фронт с <kbd>VITE_FLEET_SOURCE=api</kbd> и войдите — инструкция в
        web/README.md.
      </p>
    </main>
  )
}

// QueryView показывает загрузку, ошибку или пустой результат, а данные отдаёт children
export function QueryView<T>({
  query,
  empty,
  isEmpty = (data) => Array.isArray(data) && data.length === 0,
  children,
}: {
  query: UseQueryResult<T>
  empty?: ReactNode
  isEmpty?: (data: T) => boolean
  children: (data: T) => ReactNode
}) {
  if (query.isPending) return <p className="empty" role="status">Загружаю…</p>
  if (query.isError) {
    return (
      <div className="empty" role="alert">
        Не удалось загрузить: {errorText(query.error)}.{' '}
        <button type="button" className="btn btn-sm" onClick={() => void query.refetch()}>
          Повторить
        </button>
      </div>
    )
  }
  if (empty && isEmpty(query.data)) return <p className="empty">{empty}</p>
  return <>{children(query.data)}</>
}

// Status — лампа и подпись: цвет задаёт data-tone или data-cond из styles.css
export function Status({ tone, cond, children }: { tone?: Tone; cond?: string; children: ReactNode }) {
  return (
    <span className="status" data-tone={tone} data-cond={cond}>
      <span className="lamp" aria-hidden />
      {children}
    </span>
  )
}

// Pager — листание по limit/offset. Общего числа записей бэк не отдаёт, поэтому только «назад» и «дальше»
export function Pager({
  offset,
  pageSize,
  shown,
  hasNext,
  onChange,
}: {
  offset: number
  pageSize: number
  shown: number
  hasNext: boolean
  onChange: (offset: number) => void
}) {
  if (offset === 0 && !hasNext) return null
  return (
    <nav className="pager" aria-label="Страницы">
      <button
        type="button"
        className="btn btn-sm"
        disabled={offset === 0}
        onClick={() => onChange(Math.max(0, offset - pageSize))}
      >
        Назад
      </button>
      <span className="pager-range">
        {shown > 0 ? `${offset + 1}–${offset + shown}` : 'пусто'}
      </span>
      <button type="button" className="btn btn-sm" disabled={!hasNext} onClick={() => onChange(offset + pageSize)}>
        Дальше
      </button>
    </nav>
  )
}
