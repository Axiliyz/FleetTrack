import { useState } from 'react'
import type { Driver } from '../../types'
import { ConfirmDialog, FormDialog } from '../../ui/Dialog'
import { useCreateDriver, useDeleteDriver, useRenameDriver } from './api'

// имя водителя на бэке — VARCHAR(35)
const NAME_MAX = 35

export function CreateDriverDialog({ onClose }: { onClose: () => void }) {
  const create = useCreateDriver()
  const [name, setName] = useState('')

  return (
    <FormDialog
      title="Новый водитель"
      onClose={onClose}
      onSubmit={() => create.mutate(name.trim(), { onSuccess: onClose })}
      submitLabel="Добавить"
      pending={create.isPending}
      error={create.error}
    >
      <label className="field">
        <span>Имя и фамилия</span>
        <input required maxLength={NAME_MAX} value={name} onChange={(e) => setName(e.target.value)} />
      </label>
    </FormDialog>
  )
}

export function RenameDriverDialog({ driver, onClose }: { driver: Driver; onClose: () => void }) {
  const rename = useRenameDriver()
  const [name, setName] = useState(driver.name)

  const submit = () => {
    const next = name.trim()
    if (next === driver.name) onClose()
    else rename.mutate({ id: driver.id, name: next }, { onSuccess: onClose })
  }

  return (
    <FormDialog
      title="Переименовать водителя"
      onClose={onClose}
      onSubmit={submit}
      submitLabel="Сохранить"
      pending={rename.isPending}
      error={rename.error}
    >
      <label className="field">
        <span>Имя и фамилия</span>
        <input required maxLength={NAME_MAX} value={name} onChange={(e) => setName(e.target.value)} />
      </label>
    </FormDialog>
  )
}

export function DeleteDriverDialog({
  driver,
  onClose,
  onDeleted,
}: {
  driver: Driver
  onClose: () => void
  onDeleted?: () => void
}) {
  const remove = useDeleteDriver()
  return (
    <ConfirmDialog
      title="Удалить водителя"
      onClose={onClose}
      onConfirm={() => remove.mutate(driver.id, { onSuccess: onDeleted ?? onClose })}
      confirmLabel="Удалить"
      pending={remove.isPending}
      error={remove.error}
    >
      {driver.name} пропадёт из списка. Водителя с рейсами бэк удалить не даст — историю рейсов терять нельзя.
    </ConfirmDialog>
  )
}
