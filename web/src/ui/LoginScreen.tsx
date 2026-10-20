import { useState, type FormEvent } from 'react'
import { ApiError, auth } from '../api/client'

// LoginScreen — вход диспетчера; после успешного входа App сам подключится к парку
export function LoginScreen() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setPending(true)
    setError(null)
    try {
      await auth.login(email.trim(), password)
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : 'Сервер не отвечает. Проверьте, что API запущен на localhost:8080',
      )
      setPending(false)
    }
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

        {import.meta.env.DEV && (
          <p className="login-hint">
            Нет учётной записи? Запустите генератор парка <kbd>npm run sim</kbd>: он создаст организацию, машины и
            водителей и напишет, с какими данными войти.
          </p>
        )}
      </form>
    </main>
  )
}
