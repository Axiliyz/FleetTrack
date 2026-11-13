import { useState } from 'react'
import { formatDateTime, formatDecimal } from '../../lib/format'
import { SEVERITY_LABEL, SEVERITY_TONE, optionsOf } from '../../lib/labels'
import { ALERT_LABEL } from '../../state/selectors'
import { useAccess } from '../../state/session'
import type { AlertRule, AlertType, Severity } from '../../types'
import { ConfirmDialog, FormDialog } from '../../ui/Dialog'
import { Page, QueryView, Status } from '../../ui/Page'
import { useCreateRule, useDeleteRule, useRules } from './api'
import { AlertTabs } from './AlertTabs'

// RULE_KIND — как читать порог правила каждого типа: единицы и направление сравнения на бэке
const RULE_KIND: Record<AlertType, { unit: string; when: string; example: number }> = {
  SPEED_EXCEEDED: { unit: 'км/ч', when: 'скорость не ниже', example: 90 },
  LOW_FUEL: { unit: '%', when: 'топливо не выше', example: 15 },
  DEVICE_OFFLINE: { unit: 'мин', when: 'трекер молчит дольше', example: 5 },
}

// название правила на бэке — VARCHAR(40)
const NAME_MAX = 40

export function RulesPage() {
  const { manageFleet } = useAccess()
  const query = useRules()
  const [creating, setCreating] = useState(false)
  const [deleting, setDeleting] = useState<AlertRule | null>(null)

  return (
    <Page
      title="Тревоги"
      lead="Правила проверяются на каждой точке телеметрии, правило связи — по таймеру."
      actions={
        manageFleet && (
          <button type="button" className="btn btn-primary" onClick={() => setCreating(true)}>
            Новое правило
          </button>
        )
      }
    >
      <AlertTabs />

      <QueryView query={query} empty="Правил нет — тревоги подниматься не будут.">
        {(rules) => (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th scope="col">Правило</th>
                  <th scope="col">Срабатывает, когда</th>
                  <th scope="col">Критичность</th>
                  <th scope="col">Включено</th>
                  <th scope="col">Создано</th>
                  {manageFleet && <th scope="col" aria-label="Действия" />}
                </tr>
              </thead>
              <tbody>
                {rules.map((r) => (
                  <tr key={r.id}>
                    <td>
                      <b>{r.name}</b>
                      <span className="cell-note">{ALERT_LABEL[r.type]}</span>
                    </td>
                    <td>
                      {RULE_KIND[r.type].when} {formatDecimal(r.threshold)} {RULE_KIND[r.type].unit}
                    </td>
                    <td>
                      <Status tone={SEVERITY_TONE[r.severity]}>{SEVERITY_LABEL[r.severity]}</Status>
                    </td>
                    <td>
                      <Status tone={r.enabled ? 'ok' : 'info'}>{r.enabled ? 'Да' : 'Нет'}</Status>
                    </td>
                    <td className="nowrap">{formatDateTime(r.created_at)}</td>
                    {manageFleet && (
                      <td className="row-actions">
                        <button type="button" className="btn btn-sm btn-quiet" onClick={() => setDeleting(r)}>
                          Удалить
                        </button>
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </QueryView>

      {creating && <CreateRuleDialog onClose={() => setCreating(false)} />}
      {deleting && <DeleteRuleDialog rule={deleting} onClose={() => setDeleting(null)} />}
    </Page>
  )
}

function CreateRuleDialog({ onClose }: { onClose: () => void }) {
  const create = useCreateRule()
  const [type, setType] = useState<AlertType>('SPEED_EXCEEDED')
  const [name, setName] = useState('')
  const [threshold, setThreshold] = useState('')
  const [severity, setSeverity] = useState<Severity>('HIGH')
  const [enabled, setEnabled] = useState(true)
  const [problem, setProblem] = useState<string | null>(null)
  const kind = RULE_KIND[type]

  const submit = () => {
    const value = Number(threshold.trim().replace(',', '.'))
    const found = threshold.trim() === '' || !(value >= 0) ? 'Порог — число не меньше нуля' : null
    setProblem(found)
    if (!found) {
      create.mutate({ type, name: name.trim(), threshold: value, severity, enabled }, { onSuccess: onClose })
    }
  }

  return (
    <FormDialog
      title="Новое правило"
      onClose={onClose}
      onSubmit={submit}
      submitLabel="Создать"
      pending={create.isPending}
      error={problem ?? create.error}
    >
      <label className="field">
        <span>Что отслеживать</span>
        <select value={type} onChange={(e) => setType(e.target.value as AlertType)}>
          {optionsOf(ALERT_LABEL).map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      </label>
      <label className="field">
        <span>Название</span>
        <input
          required
          maxLength={NAME_MAX}
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={`${ALERT_LABEL[type]}: ${kind.example} ${kind.unit}`}
        />
      </label>
      <label className="field">
        <span>
          Порог, {kind.unit} — тревога, когда {kind.when} порога
        </span>
        <input
          inputMode="decimal"
          required
          value={threshold}
          onChange={(e) => setThreshold(e.target.value)}
          placeholder={String(kind.example)}
        />
      </label>
      <label className="field">
        <span>Критичность</span>
        <select value={severity} onChange={(e) => setSeverity(e.target.value as Severity)}>
          {optionsOf(SEVERITY_LABEL).map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      </label>
      <label className="check">
        <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
        Включить сразу
      </label>
      <p className="field-note">
        Высокая и критическая тревоги всплывают на карте со звуком. В Telegram уходят тревоги не ниже критичности,
        выбранной у канала.
      </p>
    </FormDialog>
  )
}

function DeleteRuleDialog({ rule, onClose }: { rule: AlertRule; onClose: () => void }) {
  const remove = useDeleteRule()
  return (
    <ConfirmDialog
      title="Удалить правило"
      onClose={onClose}
      onConfirm={() => remove.mutate(rule.id, { onSuccess: onClose })}
      confirmLabel="Удалить"
      pending={remove.isPending}
      error={remove.error}
    >
      Правило «{rule.name}» перестанет поднимать тревоги. Уже поднятые останутся в журнале.
    </ConfirmDialog>
  )
}
