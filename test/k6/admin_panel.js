// k6 load test for the admin panel.
// Run: k6 run test/k6/admin_panel.js
//
// Install k6: brew install k6
// Requires the app running with an active auction.
// Set AUCTION_SLUG and LOT_NUM env vars, e.g.:
//   k6 run -e AUCTION_SLUG=spring-2025 -e LOT_NUM=1 test/k6/admin_panel.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const BASE = 'http://localhost:8002';
const SLUG = __ENV.AUCTION_SLUG || 'test';
const LOT  = __ENV.LOT_NUM      || '1';

const errorRate   = new Rate('errors');
const bidFeedTime = new Trend('bid_feed_duration', true);

export const options = {
  scenarios: {
    // Simulate 20 admin operators polling the bid feed
    bid_feed_polling: {
      executor: 'constant-vus',
      vus: 20,
      duration: '30s',
    },
    // Spike: sudden 100 concurrent requests to the lot detail page
    spike: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '5s',  target: 100 },
        { duration: '10s', target: 100 },
        { duration: '5s',  target: 0   },
      ],
      startTime: '35s',
    },
  },
  thresholds: {
    http_req_duration:  ['p(95)<500'],  // 95% of requests under 500ms
    http_req_failed:    ['rate<0.01'],  // less than 1% errors
    errors:             ['rate<0.01'],
  },
};

// Get a session cookie once per VU.
export function setup() {
  const res = http.post(`${BASE}/admin/login`, {
    user:     'admin',
    password: 'changeme',
  }, { redirects: 0 });
  const cookie = res.headers['Set-Cookie'];
  return { cookie };
}

export default function (data) {
  const headers = { Cookie: data.cookie };

  // 1. Auction detail page (lot grid, refreshed every 5s by htmx)
  {
    const res = http.get(`${BASE}/admin/auctions/${SLUG}`, { headers });
    check(res, { 'auction detail 200': r => r.status === 200 });
    errorRate.add(res.status !== 200);
  }

  // 2. Bid feed partial (htmx polls this every 3s)
  {
    const start = Date.now();
    const res = http.get(
      `${BASE}/admin/auctions/${SLUG}/lots/${LOT}/bids`,
      { headers }
    );
    bidFeedTime.add(Date.now() - start);
    check(res, { 'bid feed 200': r => r.status === 200 });
    errorRate.add(res.status !== 200);
  }

  // 3. Lot detail page
  {
    const res = http.get(
      `${BASE}/admin/auctions/${SLUG}/lots/${LOT}`,
      { headers }
    );
    check(res, { 'lot detail 200': r => r.status === 200 });
    errorRate.add(res.status !== 200);
  }

  sleep(1);
}
