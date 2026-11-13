import { lazy, Suspense, type ReactNode } from 'react'
import { QueryClientProvider } from '@tanstack/react-query'
import { createBrowserRouter, Link, Navigate, Outlet, RouterProvider, useLocation } from 'react-router'
import { queryClient } from './api/queryClient'
import { LIVE } from './config'
import { AlertsPage } from './sections/alerts/AlertsPage'
import { RulesPage } from './sections/alerts/RulesPage'
import { DevicesPage } from './sections/devices/DevicesPage'
import { DriverPage } from './sections/drivers/DriverPage'
import { DriversPage } from './sections/drivers/DriversPage'
import { TeamPage } from './sections/team/TeamPage'
import { TripPage } from './sections/trips/TripPage'
import { TripsPage } from './sections/trips/TripsPage'
import { VehiclePage } from './sections/vehicles/VehiclePage'
import { VehiclesPage } from './sections/vehicles/VehiclesPage'
import { useSession } from './state/session'
import { LoginScreen } from './ui/LoginScreen'
import { Page } from './ui/Page'
import { RegisterScreen } from './ui/RegisterScreen'
import { SideNav } from './ui/SideNav'

// карта с deck.gl — самая тяжёлая часть фронта: грузится, только когда её открыли
const MapPage = lazy(() => import('./ui/Dispatch').then((m) => ({ default: m.MapPage })))

const router = createBrowserRouter([
  { path: '/login', element: <PublicOnly><LoginScreen /></PublicOnly> },
  { path: '/register', element: <PublicOnly><RegisterScreen /></PublicOnly> },
  {
    element: <Shell />,
    children: [
      { index: true, element: <MapPage /> },
      { path: 'vehicles', element: <VehiclesPage /> },
      { path: 'vehicles/:id', element: <VehiclePage /> },
      { path: 'drivers', element: <DriversPage /> },
      { path: 'drivers/:id', element: <DriverPage /> },
      { path: 'devices', element: <DevicesPage /> },
      { path: 'trips', element: <TripsPage /> },
      { path: 'trips/:id', element: <TripPage /> },
      { path: 'alerts', element: <AlertsPage /> },
      { path: 'alerts/rules', element: <RulesPage /> },
      { path: 'team', element: <TeamPage /> },
      { path: '*', element: <NotFound /> },
    ],
  },
])

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}

// Shell — рабочее место после входа: разделы слева, раздел справа.
// Без сессии отправляет на вход и запоминает, куда человек шёл
function Shell() {
  const session = useSession()
  const location = useLocation()
  if (LIVE && !session) return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />
  return (
    <div className="shell">
      <SideNav />
      <div className="shell-main">
        <Suspense fallback={<p className="overlay">Загружаю…</p>}>
          <Outlet />
        </Suspense>
      </div>
    </div>
  )
}

// PublicOnly — вход и регистрация; кто уже вошёл (или работает с симулятором), идёт дальше
function PublicOnly({ children }: { children: ReactNode }) {
  const session = useSession()
  const location = useLocation()
  if (!LIVE || session) {
    const from = (location.state as { from?: string } | null)?.from
    return <Navigate to={from ?? '/'} replace />
  }
  return <>{children}</>
}

function NotFound() {
  return (
    <Page title="Нет такой страницы" needsApi={false}>
      <p className="empty">
        Адрес устарел или набран с ошибкой. <Link to="/">На главную</Link>
      </p>
    </Page>
  )
}
