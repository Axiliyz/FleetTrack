import http from 'k6/http'
import { check } from 'k6'
import { scenarioOptions, thresholds } from './profile.js'

export const options = {
    scenarios: { ramp: scenarioOptions },
    thresholds,
}

const user = {
    email: 'loadtest@example.com',
    password: 'loadtest-pass-123',
}

export function setup() {
    const res = http.post(`${__ENV.BASE_URL}/login`, JSON.stringify(user), {
        headers: { 'Content-Type': 'application/json' },
    })
    return res.json().data.access_token
}

export default function (token) {
    const res = http.get(`${__ENV.BASE_URL}/trips`, { headers: { Authorization: `Bearer ${token}` } })
    check(res, { 'trips list ok': (x) => x.status === 200 })
}
