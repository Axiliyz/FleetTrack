import http from 'k6/http'
import { check } from 'k6'
import { scenarioOptions, thresholds } from './profile.js'

export const options = {
    scenarios: { ramp: scenarioOptions },
    thresholds,
}

const pairs = JSON.parse(open('./pairs.json'))

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
    const pair = pairs[Math.floor(Math.random() * pairs.length)]
    const body = {
        device_id: pair.device_id,
        vehicle_id: pair.vehicle_id,
        lat: 55.7 + (pair.vehicle_id % 100) * 0.001 + Math.random() * 0.02,
        lon: 37.7 + (pair.vehicle_id % 100) * 0.001 + Math.random() * 0.02,
        fuel: Math.random() * 97,
    }
    const res = http.post(`${__ENV.BASE_URL}/telemetry`, JSON.stringify(body), {
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    })
    check(res, { 'telemetry post ok': (r) => r.status === 201 })
}
