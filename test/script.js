// docker run --rm -i grafana/k6 run - <script.js

import http from 'k6/http';
import { URL } from 'https://jslib.k6.io/url/1.0.0/index.js';
import { sleep } from 'k6';
import exec from 'k6/execution';

export const options = {
  vus: 5,
  duration: '10s',
};


export default function () {
  const url = 'http://0.0.0.0:8000/set';

  const payload = JSON.stringify({
    key: 'test',
    value: 'test-value',
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  http.post(url, payload, params);

  sleep(1);
}