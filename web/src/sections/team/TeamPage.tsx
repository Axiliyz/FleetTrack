import { useState } from 'react'
import { PAGE_SIZE } from '../../api/query'
import { ROLE_LABEL, ROLE_NOTE, optionsOf } from '../../lib/labels'
import { useListParams } from '../../lib/useListParams'
import { passwordProblem } from '../../lib/validate'
import { useAccess, useClaims } from '../../state/session'
import type { User, UserRole } from '../../types'
import { ConfirmDialog, FormDialog } from '../../ui/Dialog'
import { FilterBar, FilterSelect } from '../../ui/Filters'
import { Page, Pager, QueryView } from '../../ui/Page'
import { useDriverIndex } from '../drivers/api'
import { USER_FILTER_KEYS, useCreateUser, useDeleteUser, useOrganization, useUserPage } from './api'

// водитель заводится с привязкой к карточке водителя: без неё роль ничего не видит
const CREATABLE_ROLES: UserRole[] = ['DISPATCHER', 'ANALYTIC', 'DRIVER', 'ADMIN']

export function TeamPage() {
  const { admin } = useAccess()
  const claims = useClaims()
  const org = useOrganization()
  const list = useListParams(USER_FILTER_KEYS)
  const query = useUserPage(list.filter, list.offset)
  const [creating, setCreating] = useState(false)
  const [deleting, setDeleting] = useState<User | null>(null)

  return (
    <Page
      title="Люди"
      lead={org.data ? `Учётные записи организации «${org.data.name}».` : 'Учётные записи организации.'}
      actions={
        admin && (
          <button type="button" className="btn btn-primary" onClick={() => setCreating(true)}>
            Добавить человека
          </button>
        )
      }
    >
      <FilterBar active={list.active} onReset={list.reset}>
        <FilterSelect label="Роль" value={list.filter.role} onChange={(v) => list.set('role', v)} options={optionsOf(ROLE_LABEL)} />
      </FilterBar>

      <QueryView
        query={query}
        isEmpty={(page) => page.rows.length === 0 && list.offset === 0}
        empty="Никого с такой ролью."
      >
        {(page) => (
          <>
            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th scope="col">Имя</th>
                    <th scope="col">Почта</th>
                    <th scope="col">Роль</th>
                    {admin && <th scope="col" aria-label="Действия" />}
                  </tr>
                </thead>
                <tbody>
                  {page.rows.map((u) => {
                    const self = u.id === claims?.user_id
                    return (
                      <tr key={u.id}>
                        <td>
                          {u.name}
                          {self && <span className="cell-note">это вы</span>}
                        </td>
                        <td>{u.email}</td>
                        <td>
                          {ROLE_LABEL[u.role]}
                          <span className="cell-note">{ROLE_NOTE[u.role]}</span>
                        </td>
                        {admin && (
                          <td className="row-actions">
                            {!self && (
                              <button type="button" className="btn btn-sm btn-quiet" onClick={() => setDeleting(u)}>
                                Удалить
                              </button>
                            )}
                          </td>
                        )}
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
            <Pager
              offset={list.offset}
              pageSize={PAGE_SIZE}
              shown={page.rows.length}
              hasNext={page.hasNext}
              onChange={list.setOffset}
            />
          </>
        )}
      </QueryView>

      {creating && <CreateUserDialog onClose={() => setCreating(false)} />}
      {deleting && <DeleteUserDialog user={deleting} onClose={() => setDeleting(null)} />}
    </Page>
  )
}

function CreateUserDialog({ onClose }: { onClose: () => void }) {
  const create = useCreateUser()
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState<UserRole>('DISPATCHER')
  const [driverId, setDriverId] = useState('')
  const [problem, setProblem] = useState<string | null>(null)
  const drivers = useDriverIndex()
  const isDriver = role === 'DRIVER'

  const submit = () => {
    const user = {
      name: name.trim(),
      email: email.trim(),
      password,
      role,
      // бэк требует driver_id у водителя и не принимает его у остальных ролей
      ...(isDriver && { driver_id: Number(driverId) }),
    }
    const found = isDriver && !driverId ? 'выберите карточку водителя' : passwordProblem(user.password, user.email, user.name)
    setProblem(found)
    if (!found) create.mutate(user, { onSuccess: onClose })
  }

  return (
    <FormDialog
      title="Новый человек"
      onClose={onClose}
      onSubmit={submit}
      submitLabel="Добавить"
      pending={create.isPending}
      error={problem ?? create.error}
    >
      <label className="field">
        <span>Имя</span>
        <input required maxLength={35} value={name} onChange={(e) => setName(e.target.value)} />
      </label>
      <label className="field">
        <span>Почта — это логин</span>
        <input type="email" required maxLength={100} value={email} onChange={(e) => setEmail(e.target.value)} />
      </label>
      <label className="field">
        <span>Временный пароль</span>
        <input
          type="password"
          required
          minLength={8}
          maxLength={72}
          autoComplete="new-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
      </label>
      <label className="field">
        <span>Роль</span>
        <select value={role} onChange={(e) => setRole(e.target.value as UserRole)}>
          {CREATABLE_ROLES.map((r) => (
            <option key={r} value={r}>
              {ROLE_LABEL[r]} — {ROLE_NOTE[r]}
            </option>
          ))}
        </select>
      </label>
      {isDriver && (
        <label className="field">
          <span>Карточка водителя</span>
          <select required value={driverId} onChange={(e) => setDriverId(e.target.value)}>
            <option value="">—</option>
            {Object.values(drivers.data ?? {}).map((d) => (
              <option key={d.id} value={d.id}>
                {d.name}
              </option>
            ))}
          </select>
        </label>
      )}
    </FormDialog>
  )
}

function DeleteUserDialog({ user, onClose }: { user: User; onClose: () => void }) {
  const remove = useDeleteUser()
  return (
    <ConfirmDialog
      title="Удалить учётную запись"
      onClose={onClose}
      onConfirm={() => remove.mutate(user.id, { onSuccess: onClose })}
      confirmLabel="Удалить"
      pending={remove.isPending}
      error={remove.error}
    >
      {user.name} ({user.email}) больше не сможет войти.
    </ConfirmDialog>
  )
}
