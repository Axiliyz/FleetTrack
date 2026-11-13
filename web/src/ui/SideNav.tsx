import { useQuery } from '@tanstack/react-query'
import { NavLink } from 'react-router'
import { API_BASE, auth } from '../api/client'
import { LIVE } from '../config'
import { ROLE_LABEL } from '../lib/labels'
import type { Access } from '../lib/access'
import { useAccess, useClaims, useSession } from '../state/session'

interface Section {
  to: string
  label: string
  // кому раздел виден; без условия — всем, кто вошёл
  when?: (a: Access) => boolean
}

// SECTIONS — разделы в порядке работы диспетчера: карта, парк, рейсы, тревоги, люди
export const SECTIONS: Section[] = [
  { to: '/', label: 'Карта', when: (a) => a.viewFleet },
  { to: '/vehicles', label: 'Машины' },
  { to: '/drivers', label: 'Водители' },
  { to: '/devices', label: 'Трекеры', when: (a) => a.manageFleet },
  { to: '/trips', label: 'Рейсы' },
  { to: '/alerts', label: 'Тревоги' },
  { to: '/team', label: 'Люди' },
]

export function SideNav() {
  const access = useAccess()
  return (
    <nav className="sidenav" aria-label="Разделы">
      <span className="sidenav-brand" aria-hidden>
        <span className="brand-mark" />
      </span>
      <ul className="sidenav-links">
        {SECTIONS.filter((s) => !s.when || s.when(access)).map((s) => (
          <li key={s.to}>
            <NavLink to={s.to} end={s.to === '/'} className="sidenav-link">
              {s.label}
            </NavLink>
          </li>
        ))}
      </ul>
      {LIVE && <Account />}
    </nav>
  )
}

function Account() {
  const session = useSession()
  const claims = useClaims()
  const ready = useReadiness()

  return (
    <div className="sidenav-foot">
      <p className="sidenav-health" data-tone={ready === null ? 'info' : ready ? 'ok' : 'alarm'} role="status">
        <span className="lamp" aria-hidden />
        {ready === null ? 'API…' : ready ? 'API в норме' : 'API недоступен'}
      </p>
      <a className="sidenav-doc" href={`${API_BASE}/swagger/index.html`} target="_blank" rel="noreferrer">
        Swagger
      </a>
      {session && (
        <p className="sidenav-user" title={session.email}>
          {claims ? ROLE_LABEL[claims.role] : session.email}
        </p>
      )}
      <button type="button" className="btn btn-sm btn-quiet" onClick={() => void auth.logout()}>
        Выйти
      </button>
    </div>
  )
}

// useReadiness опрашивает /readyz: готов ли сервис принимать запросы (база на месте).
// null — ещё не знаем
function useReadiness(): boolean | null {
  const query = useQuery({
    queryKey: ['readyz'],
    queryFn: async () => (await fetch(`${API_BASE}/readyz`)).ok,
    refetchInterval: 30_000,
    retry: false,
  })
  if (query.isPending) return null
  return query.data === true
}
