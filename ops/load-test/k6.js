// k6 load test for the public read path. Run with:
//   k6 run --vus 50 --duration 60s ops/load-test/k6.js
//
// The targets here mirror the perf budget in docs/ARCHITECTURE.md §16:
//   * p95 read latency ≤ 200 ms
//   * 200 concurrent read RPS sustained
//
// Set BASE_URL to the environment under test:
//   k6 run -e BASE_URL=https://api.tansiq.org ops/load-test/k6.js

import http from 'k6/http'
import { check, sleep } from 'k6'

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080'

export const options = {
  thresholds: {
    http_req_duration: ['p(95)<200'],
    http_req_failed:   ['rate<0.01'],
  },
  scenarios: {
    ramp: {
      executor: 'ramping-arrival-rate',
      startRate: 10,
      timeUnit: '1s',
      preAllocatedVUs: 50,
      maxVUs: 200,
      stages: [
        { target: 50,  duration: '15s' },
        { target: 200, duration: '30s' },
        { target: 200, duration: '30s' },
        { target: 0,   duration: '15s' },
      ],
    },
  },
}

export default function () {
  const endpoints = [
    '/health',
    '/api/v1/locations/countries',
    '/api/v1/crime-types',
    '/api/v1/cases?limit=20',
    '/api/v1/persons?limit=20',
  ]
  const url = BASE_URL + endpoints[Math.floor(Math.random() * endpoints.length)]
  const res = http.get(url)
  check(res, { 'status is 2xx': (r) => r.status >= 200 && r.status < 300 })
  sleep(0.1)
}
