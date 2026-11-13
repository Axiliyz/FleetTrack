import { useEffect, useId, useRef, useState, type FormEvent, type ReactNode } from 'react'
import { errorText } from '../api/errors'

// Dialog — модальное окно на нативном <dialog>: фокус, Esc и затемнение даёт браузер.
// Окно открыто, пока смонтировано: родитель показывает его условием и убирает в onClose.
// Так форма и состояние запроса каждый раз начинаются с чистого листа.
// busy — идёт запрос: закрыть окно нельзя, иначе пропал бы его итог (переход к созданной записи, ошибка)
export function Dialog({
  title,
  onClose,
  busy = false,
  children,
}: {
  title: string
  onClose: () => void
  busy?: boolean
  children: ReactNode
}) {
  const ref = useRef<HTMLDialogElement>(null)
  const titleId = useId()
  // окно убирают размонтированием, а фокус браузер возвращает только при close() — вернём его сами.
  // Кто открыл окно, запоминаем при первом рендере: к эффекту фокус уже может быть внутри окна
  const [opener] = useState(() => (document.activeElement instanceof HTMLElement ? document.activeElement : null))

  useEffect(() => {
    const dialog = ref.current
    if (!dialog) return
    if (!dialog.open) dialog.showModal()
    // showModal ставит фокус на первую кнопку — крестик, и пробел при наборе закрыл бы окно.
    // Ведём фокус в первое поле формы, а если полей нет — на кнопку подтверждения
    const target = dialog.querySelector<HTMLElement>('input, select, textarea, button[type="submit"]')
    target?.focus()
    // close() здесь не зовём: событие close пришло бы уже после размонтирования и закрыло бы окно,
    // которое StrictMode тут же монтирует заново
    return () => {
      if (opener?.isConnected) requestAnimationFrame(() => opener.focus())
    }
  }, [opener])

  return (
    <dialog
      ref={ref}
      className="dialog"
      aria-labelledby={titleId}
      aria-busy={busy || undefined}
      onCancel={(e) => {
        if (busy) e.preventDefault()
      }}
      onClose={onClose}
    >
      <header className="dialog-head">
        <h2 id={titleId}>{title}</h2>
        <button type="button" className="card-close" aria-label="Закрыть" disabled={busy} onClick={onClose}>
          ×
        </button>
      </header>
      {children}
    </dialog>
  )
}

// FormDialog — окно с формой: кнопки, ожидание ответа и ошибка сервера уже на месте
export function FormDialog({
  title,
  onClose,
  onSubmit,
  submitLabel,
  pending,
  error,
  danger = false,
  children,
}: {
  title: string
  onClose: () => void
  onSubmit: () => void
  submitLabel: string
  pending: boolean
  error: unknown
  danger?: boolean
  children: ReactNode
}) {
  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (!pending) onSubmit()
  }

  return (
    <Dialog title={title} onClose={onClose} busy={pending}>
      <form className="dialog-body" onSubmit={submit}>
        {children}
        {error ? <FormError error={error} /> : null}
        <div className="dialog-actions">
          <button type="button" className="btn btn-quiet" disabled={pending} onClick={onClose}>
            Отмена
          </button>
          <button type="submit" className={danger ? 'btn btn-alarm' : 'btn btn-primary'} disabled={pending}>
            {pending ? 'Сохраняю…' : submitLabel}
          </button>
        </div>
      </form>
    </Dialog>
  )
}

// ConfirmDialog — подтверждение необратимого действия вместо window.confirm
export function ConfirmDialog({
  title,
  onClose,
  onConfirm,
  confirmLabel,
  pending,
  error,
  children,
}: {
  title: string
  onClose: () => void
  onConfirm: () => void
  confirmLabel: string
  pending: boolean
  error: unknown
  children: ReactNode
}) {
  return (
    <FormDialog
      title={title}
      onClose={onClose}
      onSubmit={onConfirm}
      submitLabel={confirmLabel}
      pending={pending}
      error={error}
      danger
    >
      <p className="dialog-text">{children}</p>
    </FormDialog>
  )
}

// FormError — строка с готовым текстом или ошибка запроса, которую надо перевести
export function FormError({ error }: { error: unknown }) {
  return (
    <p className="form-error" role="alert">
      {typeof error === 'string' ? error : errorText(error)}
    </p>
  )
}
