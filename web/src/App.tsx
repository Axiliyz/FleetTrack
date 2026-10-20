import { useEffect, useSyncExternalStore } from 'react'
import { auth } from './api/client'
import { LIVE } from './config'
import { unlockAudio } from './lib/sound'
import { FleetMap } from './map/FleetMap'
import { ApiFleetSource } from './source/api/ApiFleetSource'
import { MockFleetSource } from './source/mock/MockFleetSource'
import { connect, togglePause } from './state/connect'
import { useFleet } from './state/fleet'
import { AlertToast } from './ui/AlertToast'
import { EventLog } from './ui/EventLog'
import { LoginScreen } from './ui/LoginScreen'
import { Roster } from './ui/Roster'
import { TopBar } from './ui/TopBar'
import { UnitCard } from './ui/UnitCard'


const useSession = () => useSyncExternalStore(auth.onChange, auth.current)

export function App() {
  const session = useSession()
  if (LIVE && !session) return <LoginScreen />
  // key: новая сессия — новое подключение и чистый стор
  return <Dispatch key={session?.email ?? 'mock'} />
}

function Dispatch() {
  const status = useFleet((s) => s.status)
  const error = useFleet((s) => s.error)

  useEffect(() => {
    useFleet.getState().reset()
    return connect(LIVE ? new ApiFleetSource() : new MockFleetSource())
  }, [])
  useHotkeys()

  return (
    <div className="app" onPointerDown={unlockAudio}>
      <TopBar />
      <Roster />
      <main className="board">
        <FleetMap />
        <EventLog />
        <UnitCard />
        <AlertToast />
        {status !== 'ready' && (
          <div className="overlay" role="status">
            {status === 'loading' ? 'Подключаюсь к парку…' : `Не удалось загрузить парк: ${error}`}
          </div>
        )}
      </main>
    </div>
  )
}

function useHotkeys() {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement | null
      if (e.altKey || e.ctrlKey || e.metaKey || target?.closest('input, textarea, select')) return
      const store = useFleet.getState()

      // пробел на кнопке нажимает её, поэтому там пауза только по P
      if (e.code === 'KeyP' || (e.code === 'Space' && !target?.closest('button'))) {
        e.preventDefault()
        togglePause()
      } else if (e.key === 'Escape') {
        store.select(null)
        store.setRosterOpen(false)
      } else if (e.code === 'KeyF' && store.selectedId !== null) {
        store.setFollow(store.followId === store.selectedId ? null : store.selectedId)
      } else if (e.code === 'BracketLeft' || e.code === 'BracketRight') {
        const ids = Object.keys(store.vehicles)
          .map(Number)
          .sort((a, b) => a - b)
        if (ids.length === 0) return
        const at = store.selectedId === null ? -1 : ids.indexOf(store.selectedId)
        const step = e.code === 'BracketRight' ? 1 : -1
        const next = ids[(at + step + ids.length) % ids.length]
        store.select(next, { fly: true })
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])
}
