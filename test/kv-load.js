// docker run -v $(pwd):/scripts --workdir "/scripts" --network="host" -it --rm grafana/k6:latest run kv-load.js
// With dashboard: docker run -v $(pwd):/scripts --workdir "/scripts" --network="host" -p 5665:5665 -it --rm ghcr.io/szkiba/xk6-dashboard:latest run --out=dashboard kv-load.js  

import http from 'k6/http';
import { sleep, group } from 'k6';
import { URL } from 'https://jslib.k6.io/url/1.0.0/index.js';


export const options = {
  vus: 30,
  duration: '3m',
  gracefulStop: '3s',
  summaryTrendStats: ["med", "p(90)", "p(95)", "p(99)"],
};

function generateKey(vuID, id) {
  return `key-${vuID}-${id}`;
}

export const setRequests = () => {
  group('Set Requests', () => {
    for (let kd = 0; kd <= 150; kd++) {
      const key = generateKey(__VU, kd);

      const url = 'http://127.0.0.1:8000/set';
      const payload = JSON.stringify({
        key: key,
        value: `test-value-${kd}`,
      });

      const params = {
        headers: {
          'Content-Type': 'application/json',
        },
      };

      const res = http.post(url, payload, params, {
        tags: { name: 'post' },
      });

      if (res.status === 200) {
        console.log(`Set request successful for key: ${key}`);
      } else {
        console.error(`Set request failed for key: ${key}`);
      }

      sleep(0.2);
    }
  });
};

export const getRequests = () => {
  group('Get Requests', () => {
    for (let id = 0; id <= 150; id++) {
      const key = generateKey(__VU, id);

      const url = new URL('http://127.0.0.1:8000/get');
      url.searchParams.append('key', `${key}`);
      const res = http.get(url.toString(), {
        tags: { name: 'get' },
      });

      if (res.status === 200) {
        console.log(`Get request successful for key: ${key}, value: ${res.json().value}`);
      } else {
        console.error(`Get request failed for key: ${key}`);
      }

      sleep(0.1);
    }
  });
};

export default function () {
  setRequests();
  getRequests();
}