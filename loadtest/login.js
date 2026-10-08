import http from 'k6/http'
import { check } from 'k6'

export const options = {
    scenarios: {
        login_ramp: {
            executor: 'constant-arrival-rate',
            rate: 2,
            timeUnit: '1s',
            duration: '1m',
            preAllocatedVUs: 10,
            maxVUs: 50,
        },
    },
    thresholds: {
        http_req_duration: ['p(95)<500'],
    },
}

const user = {
    email: 'loadtest@example.com',
    password: 'loadtest-pass-123',
}

export default function () {
    const res = http.post(`${__ENV.BASE_URL}/login`, JSON.stringify(user), {
        headers: { 'Content-Type': 'application/json' },
    })
    check(res, {
        'login ok or rate limited': (x) => x.status === 200 || x.status === 429,
    })
}
