import http from 'k6/http';
import { check, sleep } from 'k6';

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
  stages: [
    { duration: '10s', target: 20 },   // ramp up to 20 users
    { duration: '30s', target: 20 },   // hold 20 users
    { duration: '10s', target: 0 },    // ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],   // 95% of requests should be under 500ms
  },
};

// Setup: Fetch a real merchant UUID before VUs start
export function setup() {
  const res = http.get(`${BASE_URL}/api/merchants`);
  const body = JSON.parse(res.body);

  // The merchants endpoint returns an array of merchants
  let merchants = [];
  if (body.merchants) {
    merchants = body.merchants;
  } else if (body.data) {
    merchants = body.data;
  } else if (Array.isArray(body)) {
    merchants = body;
  }

  if (!merchants || merchants.length === 0) {
    throw new Error('No merchants found. Is the backend seeded?');
  }

  const merchant = merchants[0];
  const merchantName = merchant.MerchantName || merchant.merchant_name || 'Unknown';
  const merchantID = merchant.ID || merchant.id;
  console.log(`Using merchant: ${merchantName} (${merchantID})`);

  return { merchantID: merchantID };
}

// Main test: Generate QRIS
export default function (data) {
  const amount = Math.floor(Math.random() * 100000) + 1000; // 1000 - 101000

  const res = http.get(
    `${BASE_URL}/api/qris?merchant_id=${data.merchantID}&amount=${amount}`
  );

  check(res, {
    'status is 200': (r) => r.status === 200,
    'has qris_payload': (r) => JSON.parse(r.body).qris_payload !== undefined,
  });

  sleep(0.5);
}
