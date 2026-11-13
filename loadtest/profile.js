// SHORT=1 - короткий прогон (~3 мин, до 400 запросов/с) для сравнения двух версий на одной машине;
// без него идёт полный профиль до 2500 запросов/с
const short = __ENV.SHORT === '1'

const fullStages = [
    { duration: '30s', target: 500 },
    { duration: '2m', target: 500 },
    { duration: '30s', target: 1000 },
    { duration: '2m', target: 1000 },
    { duration: '30s', target: 2000 },
    { duration: '2m', target: 2000 },
    { duration: '30s', target: 2500 },
    { duration: '2m', target: 2500 },
    { duration: '30s', target: 0 },
]

const shortStages = [
    { duration: '20s', target: 200 },
    { duration: '60s', target: 200 },
    { duration: '20s', target: 400 },
    { duration: '60s', target: 400 },
    { duration: '10s', target: 0 },
]

export const scenarioOptions = {
    executor: 'ramping-arrival-rate',
    startRate: 10,
    timeUnit: '1s',
    preAllocatedVUs: 100,
    maxVUs: 2000,
    stages: short ? shortStages : fullStages,
}

export const thresholds = {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.01'],
}
