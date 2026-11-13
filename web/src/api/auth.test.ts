import { afterEach, describe, expect, it, vi } from 'vitest'
import { json, recordFetch } from '../test/http'
import { auth, decodeClaims } from './client'

// токен в формате JWT: подпись фронт не проверяет, поэтому она тут фиктивная
const token = (payload: object) =>
  ['header', Buffer.from(JSON.stringify(payload)).toString('base64url'), 'signature'].join('.')

afterEach(async () => {
  recordFetch()
  await auth.logout()
  vi.unstubAllGlobals()
})

describe('decodeClaims', () => {
  it('читает роль и организацию, в том числе с кириллицей в base64url', () => {
    const claims = decodeClaims(token({ user_id: 1, organization_id: 5, driver_id: null, role: 'DISPATCHER', exp: 1, note: 'ё' }))
    expect(claims).toMatchObject({ user_id: 1, organization_id: 5, role: 'DISPATCHER' })
  })

  it('мусор вместо токена — null, а не исключение', () => {
    expect(decodeClaims('not-a-jwt')).toBeNull()
    expect(decodeClaims('a.%%%.c')).toBeNull()
  })
})

describe('auth.register', () => {
  it('регистрирует организацию и сразу входит под администратором', async () => {
    const access = token({ user_id: 1, organization_id: 9, driver_id: null, role: 'ADMIN', exp: 1 })
    const calls = recordFetch(() => json({ access_token: access, refresh_token: 'r-1' }, 201))

    await auth.register({
      organization_name: 'Северный тракт',
      admin_name: 'Анна',
      admin_email: 'anna@fleettrack.test',
      admin_password: 'secret12',
    })

    expect(calls[0]).toMatchObject({ method: 'POST', url: '/api/register' })
    expect(auth.current()?.email).toBe('anna@fleettrack.test')
    expect(auth.claims()?.role).toBe('ADMIN')
  })

  it('ошибка регистрации не создаёт сессию', async () => {
    recordFetch(() => json('user with this email already exists', 409))
    await expect(
      auth.register({ organization_name: 'X', admin_name: 'Y', admin_email: 'y@x.z', admin_password: 'secret12' }),
    ).rejects.toMatchObject({ status: 409 })
    expect(auth.current()).toBeNull()
  })
})
