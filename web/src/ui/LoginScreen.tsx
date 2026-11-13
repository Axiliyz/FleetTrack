import { useState, type FormEvent } from 'react'
import { Link } from 'react-router'
import { ApiError, auth } from '../api/client'
import { errorText } from '../api/errors'
import { DEMO } from '../config'

// LoginScreen — вход диспетчера; после успешного входа роутер сам вернёт туда, куда человек шёл
export function LoginScreen() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)
  const demo = DEMO

  const enter = async (login: string, secret: string) => {
    setPending(true)
    setError(null)
    try {
      await auth.login(login, secret)
    } catch (err) {
      // 401 и 429 auth.login уже перевёл; в dev подсказываем, где обычно причина недоступности
      const offline = !(err instanceof ApiError)
      setError(
        offline && import.meta.env.DEV ? 'Сервер не отвечает. Проверьте, что API запущен на localhost:8080' : errorText(err),
      )
      setPending(false)
    }
  }

  const submit = (e: FormEvent) => {
    e.preventDefault()
    void enter(email.trim(), password)
  }

  return (
    <main className="login">
      <form className="login-panel" onSubmit={submit} aria-labelledby="login-title">
        <div className="brand">
          <span className="brand-mark" aria-hidden />
          <span className="brand-name" id="login-title">
            FleetTrack
          </span>
        </div>
        <p className="login-lead">Войдите, чтобы открыть карту своего парка.</p>

        <label className="field">
          <span>Почта</span>
          <input
            type="email"
            autoComplete="username"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
        </label>
        <label className="field">
          <span>Пароль</span>
          <input
            type="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </label>

        {error && (
          <p className="login-error" role="alert">
            {error}
          </p>
        )}

        <button type="submit" className="btn btn-primary" disabled={pending}>
          {pending ? 'Вхожу…' : 'Войти'}
        </button>

        {demo && (
          <button type="button" className="btn" disabled={pending} onClick={() => void enter(demo.email, demo.password)}>
            Войти в демо-парк
          </button>
        )}

        <p className="login-hint">
          Своего парка ещё нет? <Link to="/register">Заведите организацию</Link>
          {import.meta.env.DEV && (
            <>
              {' '}
              или запустите генератор <kbd>npm run sim</kbd>: он создаст организацию, машины и водителей и напишет, с
              какими данными войти.
            </>
          )}
        </p>
      </form>
    </main>
  )
}
