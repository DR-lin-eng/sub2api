import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 2,
  duration: '15s',
  thresholds: {
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<5000'],
  },
};

const baseURL = __ENV.TARGET_URL || 'http://host.docker.internal:8080';
const apiKey = __ENV.API_KEY || '';

export default function () {
  const payload = JSON.stringify({
    model: 'claude-sonnet-4-5',
    stream: false,
    max_tokens: 32,
    messages: [{ role: 'user', content: 'ping' }],
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${apiKey}`,
    },
  };

  const res = http.post(`${baseURL}/v1/messages`, payload, params);
  check(res, {
    'status is 2xx/4xx/5xx': (r) => r.status >= 200,
  });
  sleep(1);
}
