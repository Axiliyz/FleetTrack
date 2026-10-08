import http from 'k6/http'
import { check } from 'k6'
import { scenarioOptions, thresholds } from './profile.js'

export const options = {
    scenarios: { ramp: scenarioOptions },
    thresholds,
}

export default function () {
    const res = http.get(`${__ENV.BASE_URL}/health`)
    check(res, { healthy: (r) => r.status === 200 })
}
