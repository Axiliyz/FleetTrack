import { useState, type ChangeEvent, type FormEvent } from 'react'
import { Link } from 'react-router'
import { auth, type Registration } from '../api/client'
import { errorText } from '../api/errors'
import { passwordProblem } from '../lib/validate'
import { FormError } from './Dialog'

const EMPTY: Registration = { organization_name: '', admin_name: '', admin_email: '', admin_password: '' }

// RegisterScreen — новая организация с первым администратором. После регистрации человек сразу внутри
export function RegisterScreen() {
  const [form, setForm] = useState(EMPTY)
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  const field = (key: keyof Registration) => ({
    value: form[key],
    onChange: (e: ChangeEvent<HTMLInputElement>) => setForm({ ...form, [key]: e.target.value }),
  })

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    const clean = {
      ...form,
      organization_name: form.organization_name.trim(),
      admin_name: form.admin_name.trim(),
      admin_email: form.admin_email.trim(),
    }
    const problem = passwordProblem(clean.admin_password, clean.admin_email, clean.admin_name)
    if (problem) {
      setError(problem)
      return
    }
    setPending(true)
    setError(null)
    try {
      await auth.register(clean)
    } catch (err) {
      setError(errorText(err))
      setPending(false)
    }
  }

  return (
    <main className="login">
      <form className="login-panel" onSubmit={submit} aria-labelledby="register-title">
        <div className="brand">
          <span className="brand-mark" aria-hidden />
          <span className="brand-name" id="register-title">
            Новая организация
          </span>
        </div>
        <p className="login-lead">Вы станете её администратором и сможете завести машины, водителей и коллег.</p>

        <label className="field">
          <span>Название организации</span>
          <input required maxLength={255} autoComplete="organization" {...field('organization_name')} />
        </label>
        <label className="field">
          <span>Ваше имя</span>
          <input required maxLength={35} autoComplete="name" {...field('admin_name')} />
        </label>
        <label className="field">
          <span>Почта</span>
          <input type="email" required maxLength={100} autoComplete="username" {...field('admin_email')} />
        </label>
        <label className="field">
          <span>Пароль</span>
          <input type="password" required minLength={8} maxLength={72} autoComplete="new-password" {...field('admin_password')} />
        </label>

        {error && <FormError error={error} />}

        <button type="submit" className="btn btn-primary" disabled={pending}>
          {pending ? 'Создаю…' : 'Создать и войти'}
        </button>

        <p className="login-hint">
          Уже есть учётная запись? <Link to="/login">Войти</Link>
        </p>
      </form>
    </main>
  )
}
