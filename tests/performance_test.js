import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate } from 'k6/metrics';
import { randomString } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// Custom metrics
const errorRate = new Rate('errors');

// Environment variables
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8085/api/v1';
const USER_EMAIL_PREFIX = __ENV.USER_EMAIL_PREFIX || 'testuser';
const USER_PASSWORD = __ENV.USER_PASSWORD || 'Test123456';
const MAX_USERS = parseInt(__ENV.MAX_USERS) || 20; // Reduced for faster setup

// Test configuration
export const options = {
  setupTimeout: '120s', // Increased timeout
  scenarios: {
    load_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 50 },
        { duration: '5m', target: 50 },
        { duration: '2m', target: 0 },
      ],
      gracefulRampDown: '30s',
    },
    stress_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 100 },
        { duration: '3m', target: 200 },
        { duration: '2m', target: 0 },
      ],
      gracefulRampDown: '30s',
      startTime: '10m',
    },
    soak_test: {
      executor: 'constant-vus',
      vus: 30,
      duration: '30m',
      startTime: '17m',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<700', 'p(99)<2000'], // Adjusted thresholds
    'errors': ['rate<0.01'],
    'http_req_failed': ['rate<0.01'],
    'checks': ['rate>0.99'],
  },
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(95)', 'p(99)'],
};

// Shared data
let tokens = [];
let productIds = [];
let registeredUsers = [];

// Setup: Register users and fetch products
export function setup() {
  console.log('Starting setup...');
  const setupData = {
    tokens: [],
    productIds: [],
    registeredUsers: [],
  };

  // Parallelize user registration and login
  const registerRequests = [];
  for (let i = 0; i < MAX_USERS; i++) {
    const email = `${USER_EMAIL_PREFIX}_${randomString(8)}@example.com`;
    const registerPayload = {
      email: email,
      name: `Test User ${i}`,
      password: USER_PASSWORD,
    };
    registerRequests.push([
      'POST',
      `${BASE_URL}/users/register`,
      JSON.stringify(registerPayload),
      { headers: { 'Content-Type': 'application/json' } },
    ]);
    setupData.registeredUsers.push({ email, password: USER_PASSWORD });
  }

  console.log(`Registering ${MAX_USERS} users...`);
  const registerResponses = http.batch(registerRequests);
  registerResponses.forEach((res, i) => {
    check(res, {
      'User registered successfully': (r) => r.status === 201,
    }) || console.error(`Failed to register user ${i}: ${res.status} ${res.body}`);
  });

  // Parallelize user login
  const loginRequests = setupData.registeredUsers.map((user) => {
    const loginPayload = {
      email: user.email,
      password: user.password,
    };
    return [
      'POST',
      `${BASE_URL}/users/login`,
      JSON.stringify(loginPayload),
      { headers: { 'Content-Type': 'application/json' } },
    ];
  });

  console.log(`Logging in ${MAX_USERS} users...`);
  const loginResponses = http.batch(loginRequests);
  loginResponses.forEach((res, i) => {
    if (check(res, { 'User logged in successfully': (r) => r.status === 200 })) {
      setupData.tokens.push(res.json('token'));
    } else {
      console.error(`Failed to login user ${i}: ${res.status} ${res.body}`);
    }
  });

  // Fetch products
  console.log('Fetching products...');
  const productRes = http.get(`${BASE_URL}/products?page=1&pageSize=10`, {
    headers: { Authorization: `Bearer ${setupData.tokens[0]}` },
  });

  if (check(productRes, { 'Products fetched successfully': (r) => r.status === 200 })) {
    const products = productRes.json('Data');
    setupData.productIds = products.map((p) => p.id);
  } else {
    console.error(`Failed to fetch products: ${productRes.status} ${productRes.body}`);
  }

  console.log('Setup completed.');
  return setupData;
}

// Main test function
export default function (data) {
  tokens = data.tokens;
  productIds = data.productIds;

  if (tokens.length === 0 || productIds.length === 0) {
    console.error('Setup failed: No tokens or product IDs available.');
    return;
  }

  // Randomly select a user token
  const token = tokens[Math.floor(Math.random() * tokens.length)];
  const headers = {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${token}`,
  };

  // Group: User Flow - Browse Products
  group('Browse Products', function () {
    const res = http.get(`${BASE_URL}/products?page=1&pageSize=10`, { headers });
    check(res, {
      'Products retrieved': (r) => r.status === 200,
    }) || errorRate.add(1);
    sleep(1);
  });

  // Group: User Flow - Manage Cart
  group('Manage Cart', function () {
    let cartRes = http.get(`${BASE_URL}/carts`, { headers });
    check(cartRes, {
      'Cart retrieved': (r) => r.status === 200,
    }) || errorRate.add(1);

    if (productIds.length > 0) {
      const addItemPayload = {
        product_id: productIds[Math.floor(Math.random() * productIds.length)],
        quantity: 1,
        unit_price: 10.0,
      };
      const addItemRes = http.post(`${BASE_URL}/carts/items`, JSON.stringify(addItemPayload), { headers });
      check(addItemRes, {
        'Item added to cart': (r) => r.status === 200,
      }) || errorRate.add(1);

      const updateItemPayload = {
        product_id: addItemPayload.product_id,
        quantity: 2,
      };
      const updateItemRes = http.put(`${BASE_URL}/carts/items`, JSON.stringify(updateItemPayload), { headers });
      check(updateItemRes, {
        'Item quantity updated': (r) => r.status === 200,
      }) || errorRate.add(1);
    }
    sleep(1);
  });

  // Group: User Flow - Create Order
  group('Create Order', function () {
    if (productIds.length > 0) {
      const orderPayload = {
        customer_id: randomString(8),
        items: [
          {
            product_id: productIds[Math.floor(Math.random() * productIds.length)],
            quantity: 1,
            unit_price: 10.0,
          },
        ],
        shipping_address: {
          street: '123 Test St',
          city: 'Test City',
          state: 'TS',
          postal_code: '12345',
          country: 'Testland',
        },
      };
      const orderRes = http.post(`${BASE_URL}/orders`, JSON.stringify(orderPayload), { headers });
      check(orderRes, {
        'Order created': (r) => r.status === 201,
      }) || errorRate.add(1);

      if (orderRes.status === 201) {
        const orderId = orderRes.json('id');
        const orderDetailRes = http.get(`${BASE_URL}/orders/${orderId}`, { headers });
        check(orderDetailRes, {
          'Order details retrieved': (r) => r.status === 200,
        }) || errorRate.add(1);
      }
    }
    sleep(1);
  });

  // Group: User Flow - Initiate Payment
  group('Initiate Payment', function () {
    const paymentPayload = {
      amount: 1000,
      currency: 'USD',
      customer_id: randomString(8),
      description: 'Test Payment',
      payment_method: 'card',
      token: 'tok_visa',
    };
    const paymentRes = http.post(`${BASE_URL}/payments`, JSON.stringify(paymentPayload), { headers });
    check(paymentRes, {
      'Payment initiated': (r) => r.status === 200,
    }) || errorRate.add(1);
    sleep(1);
  });

  // Group: User Flow - Check Notifications
  group('Check Notifications', function () {
    const notificationRes = http.get(`${BASE_URL}/notifications?page=1&pageSize=10`, { headers });
    check(notificationRes, {
      'Notifications retrieved': (r) => r.status === 200,
    }) || errorRate.add(1);
    sleep(1);
  });
}

// Teardown: Cleanup test users (optional, requires admin API)
export function teardown(data) {
  console.log('Starting teardown...');
  // Note: The API spec doesn't provide a user deletion endpoint.
  // If available, implement deletion here using admin credentials.
  console.log('Teardown completed (no user deletion endpoint available).');
}