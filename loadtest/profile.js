export const scenarioOptions = {
    executor: 'ramping-arrival-rate',
    startRate: 10,
    timeUnit: '1s',
    preAllocatedVUs: 100,
    maxVUs: 2000,
    stages: [
        { duration: '30s', target: 500 },
        { duration: '2m', target: 500 },
        { duration: '30s', target: 1000 },
        { duration: '2m', target: 1000 },
        { duration: '30s', target: 2000 },
        { duration: '2m', target: 2000 },
        { duration: '30s', target: 2500 },
        { duration: '2m', target: 2500 },
        { duration: '30s', target: 0 },
    ],
}

export const thresholds = {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.01'],
}
