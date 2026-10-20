// Клиент API FleetTrack: снимает обёртку {status, message, data}, держит токены и обновляет их.
// Запросы идут на /api — в dev-режиме Vite проксирует их на Go-сервис, поэтому CORS не нужен.

export const API_BASE = '/api'

const STORAGE_KEY = 'fleettrack.session'

export interface Session {
  email: string
  accessToken: string
  refreshToken: string
}

export class ApiError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

interface TokenPair {
  access_token: string
  refresh_token: string
}

// Токены лежат в sessionStorage: переживают перезагрузку вкладки и пропадают при её закрытии.
// Бэк отдаёт refresh-токен в теле, а не в httpOnly-cookie, поэтому он доступен скриптам страницы —
// это осознанный компромисс dev-стенда
function readStored(): Session | null {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    return raw ? (JSON.parse(raw) as Session) : null
  } catch {
    return null
  }
}

let session: Session | null = readStored()
const listeners = new Set<(s: Session | null) => void>()

function setSession(next: Session | null): void {
  session = next
  try {
    if (next) sessionStorage.setItem(STORAGE_KEY, JSON.stringify(next))
    else sessionStorage.removeItem(STORAGE_KEY)
  } catch {
    // приватный режим или запрет хранилища: сессия живёт до перезагрузки
  }
  for (const fn of listeners) fn(next)
}

async function readError(res: Response): Promise<ApiError> {
  let message = res.statusText || `HTTP ${res.status}`
  try {
    const body = (await res.json()) as { message?: string }
    if (body.message) message = body.message
  } catch {
    // тело не JSON — оставляем текст статуса
  }
  return new ApiError(res.status, message)
}

async function unwrap<T>(res: Response): Promise<T> {
  if (!res.ok) throw await readError(res)
  const body = (await res.json()) as { data: T }
  return body.data
}

let refreshing: Promise<boolean> | null = null

export const auth = {
  current: (): Session | null => session,

  onChange(fn: (s: Session | null) => void): () => void {
    listeners.add(fn)
    return () => listeners.delete(fn)
  },

  async login(email: string, password: string): Promise<void> {
    const res = await fetch(`${API_BASE}/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })
    if (res.status === 401) throw new ApiError(401, 'Неверная почта или пароль')
    if (res.status === 429) throw new ApiError(429, 'Слишком много попыток. Подождите минуту и попробуйте снова')
    const tokens = await unwrap<TokenPair>(res)
    setSession({ email, accessToken: tokens.access_token, refreshToken: tokens.refresh_token })
  },

  async logout(): Promise<void> {
    const current = session
    setSession(null)
    if (!current) return
    // отзыв refresh-токена на сервере — по возможности; выйти локально важнее
    await fetch(`${API_BASE}/logout`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: current.refreshToken }),
    }).catch(() => undefined)
  },

  // refresh обновляет пару токенов. Параллельные запросы, получившие 401, ждут одно обновление:
  // refresh-токен одноразовый, второй запрос с ним же получил бы отказ и разлогинил пользователя
  refresh(): Promise<boolean> {
    if (refreshing) return refreshing
    const current = session
    if (!current) return Promise.resolve(false)
    refreshing = (async () => {
      try {
        const res = await fetch(`${API_BASE}/refresh`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: current.refreshToken }),
        })
        if (!res.ok) {
          setSession(null)
          return false
        }
        const tokens = await unwrap<TokenPair>(res)
        setSession({ ...current, accessToken: tokens.access_token, refreshToken: tokens.refresh_token })
        return true
      } catch {
        return false
      } finally {
        refreshing = null
      }
    })()
    return refreshing
  },

  authHeader(): Record<string, string> {
    return session ? { Authorization: `Bearer ${session.accessToken}` } : {}
  },
}

// api выполняет запрос с токеном и один раз повторяет его после обновления токена
export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const send = () =>
    fetch(`${API_BASE}${path}`, {
      ...init,
      headers: {
        ...(init.body ? { 'Content-Type': 'application/json' } : {}),
        ...auth.authHeader(),
        ...init.headers,
      },
    })

  let res = await send()
  if (res.status === 401 && (await auth.refresh())) res = await send()
  if (res.status === 401) {
    await auth.logout()
    throw new ApiError(401, 'Сессия истекла, войдите снова')
  }
  return unwrap<T>(res)
}
