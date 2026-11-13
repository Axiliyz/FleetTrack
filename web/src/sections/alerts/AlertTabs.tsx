import { NavLink } from 'react-router'

// AlertTabs — журнал тревог и правила, по которым они поднимаются: два адреса, одна вкладка в навигации
export function AlertTabs() {
  return (
    <nav className="tabs page-tabs" aria-label="Тревоги">
      <NavLink to="/alerts" end>
        Журнал
      </NavLink>
      <NavLink to="/alerts/rules">Правила</NavLink>
    </nav>
  )
}
