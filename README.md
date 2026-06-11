# QRIS Payment System

Full-stack QRIS payment simulation with async payment confirmation.

The project includes:

- Go backend API with Gin and clean repository/usecase layers
- Merchant dashboard built with React + Vite
- Customer scanner/payment app built with React + Vite
- PostgreSQL as the source of truth
- Redis cache for merchant lookup and transaction-status polling
- RabbitMQ queue for asynchronous payment confirmation
- Merchant WebSocket notifications for successful payments
- Prometheus + Grafana monitoring
- K6 load-test scripts

## Project Structure

```text
backend/          Go API, domain/usecase/repository code, QRIS payload logic
frontend/         Merchant dashboard, QRIS generation UI
customer-app/     Customer QR scanner and payment confirmation UI
k6/               Load-test scripts
grafana/          Provisioned dashboard and datasource config
report-purpose/   Architecture, flow, and report notes
docker-compose.yml
prometheus.yml
```

## Stack

- Go, Gin, GORM
- PostgreSQL 15
- Redis 7
- RabbitMQ management image
- React 19 + Vite
- Prometheus
- Grafana
- K6

## Architecture Summary

- PostgreSQL is the source of truth for merchants and transactions.
- The backend creates the `pgcrypto` extension, runs GORM `AutoMigrate`, and
  seeds default merchants at startup.
- Redis is an optional acceleration layer. If Redis is unavailable, the backend
  continues through PostgreSQL.
- Merchant data is warmed into Redis at startup and also cached when QRIS
  payloads are generated or QRID lookups happen.
- Transaction status uses cache-aside:
  Redis first, PostgreSQL fallback, then Redis repopulation.
- Payment confirmation publishes work to RabbitMQ and returns
  `PROCESSING` quickly. A background worker updates PostgreSQL to `SUCCESS` and
  invalidates the transaction cache.
- Successful confirmations publish merchant notifications to RabbitMQ. A
  notification worker sends them to the merchant dashboard through `/ws`.
- Prometheus records server latency, worker metrics, and cache metrics.

## Environment

Create a repo-root `.env` file before running Docker Compose. The checked-in
`.gitignore` intentionally ignores `.env`.

Example:

```env
DB_USER=user
DB_PASSWORD=user
DB_HOST=localhost
DB_PORT=5432
DB_NAME=qrisdatabase

REDIS_HOST=localhost
REDIS_PORT=6379

RABBITMQ_USER=guest
RABBITMQ_PASSWORD=guest
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672

WEBSOCKET_READ_DEADLINE=5m
WEBSOCKET_WRITE_DEADLINE=10s
WEBSOCKET_IDLE_CHECK_INTERVAL=1m
WEBSOCKET_IDLE_THRESHOLD=4m
WEBSOCKET_MAX_MESSAGE_SIZE=65536

GF_AUTH_ANONYMOUS_ENABLED=true
GF_AUTH_ANONYMOUS_ORG_ROLE=Admin
GF_SECURITY_ADMIN_USER=admin
GF_SECURITY_ADMIN_PASSWORD=12345
```

Docker Compose overrides service hostnames internally. For example, the backend
container receives `DB_HOST=db`, `REDIS_HOST=redis`, and
`RABBITMQ_HOST=rabbitmq`.

## Run With Docker Compose

From the repo root:

```bash
docker compose up -d
```

This starts:

- Nginx reverse proxy on `http://localhost` (port 80)
  - Merchant dashboard: `http://localhost/merchant/`
  - Customer app: `http://localhost/customer/`
  - Backend API: `http://localhost/` and `/api/`
- PostgreSQL on `localhost:5432`
- Redis on `localhost:6379`
- RabbitMQ management on `http://localhost:15672`
- Prometheus on `http://localhost:9090`
- Grafana on `http://localhost:3000`

Useful checks:

```bash
curl http://localhost/api/ping
curl http://localhost/api/health
curl http://localhost/api/merchants
curl http://localhost/metrics
```

## Run Apps Manually

For local development loops, start only the dependency containers you need, then
run the backend and apps on the host:

```bash
docker compose up -d db redis rabbitmq redisinsight pgadmin
```

If the full Compose stack is already running, stop the app container you want to
replace locally first, for example:

```bash
docker compose stop golang
```

Backend:

```bash
cd backend
go run ./cmd
```

Merchant dashboard:

```bash
cd frontend
npm install
npm run dev
```

Customer app:

```bash
cd customer-app
npm install
npm run dev
```

Default local app URLs (direct / no Nginx proxy):

```text
Backend:            http://localhost:8080
Merchant dashboard: http://localhost:5173/merchant/
Customer app:       http://localhost:5174/customer/
```

## Main API Routes

```text
GET  /api/ping
GET  /api/health
GET  /api/merchants
GET  /api/qris?merchant_id=<merchant_uuid>&amount=<amount>
GET  /api/transactions/:id
GET  /api/ws/status?merchant_id=<merchant_uuid>
GET  /ws?merchant_id=<merchant_uuid>
GET  /metrics
POST /api/transactions/scan
POST /api/transactions/:id/confirm
```

## Payment Flow

### 1. Merchant List

```text
GET /api/merchants
```

Returns active merchants from PostgreSQL. The merchant dashboard uses the UUID
`id` as `merchant_id` when generating QRIS payloads.

Seeded merchants:

```text
TEST001 - Kantin FILKOM UB
TEST002 - TESTING STORE
```

### 2. Generate QRIS

```text
GET /api/qris?merchant_id=<merchant_uuid>&amount=<amount>
```

The backend validates the merchant UUID and amount, loads the merchant from
PostgreSQL, caches merchant data in Redis, prefetches related merchants, and
returns a dynamic QRIS payload.

The QRIS payload includes merchant QRID in tag `26.01`, amount in tag `54`,
merchant name in tag `59`, city `MALANG`, and a CRC checksum in tag `63`.

### 3. Customer Scan

```text
POST /api/transactions/scan
```

Request:

```json
{
  "qr_payload": "<qris_payload>",
  "merchant_id": "TEST001",
  "amount": 1000
}
```

The customer app extracts QRID and amount from the scanned QRIS payload, then
sends them to the backend. The backend accepts `merchant_id` as either merchant
UUID or QRID, validates the QR CRC, verifies merchant and amount consistency,
creates a `PENDING` transaction in PostgreSQL, and caches it in Redis for 10
minutes.

### 4. Transaction Status

```text
GET /api/transactions/:id
```

The backend validates the UUID, checks Redis key `transaction:<id>`, falls back
to PostgreSQL on miss or corrupted cache, and returns the transaction response.

### 5. Async Payment Confirmation

```text
POST /api/transactions/:id/confirm
```

The backend validates the UUID, publishes `transaction_id` to RabbitMQ queue
`payment_confirmations`, and immediately returns:

```json
{
  "data": {
    "transaction_id": "<uuid>",
    "status": "PROCESSING"
  },
  "message": "payment accepted and is being processed in background"
}
```

The payment worker consumes the message, updates the transaction to `SUCCESS`,
deletes the old Redis transaction cache, and publishes a merchant notification.
A notification worker consumes that event and pushes a
`transaction_notification` message to connected merchant dashboard WebSocket
clients.

## Merchant WebSocket Notifications

The merchant dashboard connects to:

```text
GET /ws?merchant_id=<merchant_uuid>
```

When a transaction reaches `SUCCESS`, the backend publishes a notification to
RabbitMQ queue `merchant_notifications`. The notification worker sends a
`transaction_notification` message to every connected dashboard for that
merchant. If the merchant is disconnected, the WebSocket hub keeps a small
in-memory backlog and flushes it on reconnect.

Check connection state and pending notifications with:

```text
GET /api/ws/status?merchant_id=<merchant_uuid>
```

## Redis Keys

```text
merchant:<qr_id>          TTL 30 minutes
transaction:<uuid>        TTL 10 minutes
```

Redis is used for faster lookups and lower database read load. PostgreSQL
remains authoritative.

## Monitoring

Prometheus scrapes `/metrics` every 15 seconds. Grafana is provisioned with a
system health dashboard showing service uptime, request rate, error rate,
response latency percentiles, and Go runtime metrics.

The `/api/health` endpoint checks all dependencies and returns:

```json
{
  "status": "ok",
  "timestamp": "2026-06-10T23:00:00+07:00",
  "services": {
    "postgres":  { "status": "ok" },
    "redis":     { "status": "ok" },
    "rabbitmq":  { "status": "ok" }
  }
}
```

Returns HTTP `200` when all services are healthy, `503` when any is degraded.

Metrics exposed:

```text
http_requests_total
http_request_duration_seconds
```

Go runtime metrics (goroutines, heap, GC) are included automatically via the
Prometheus Go collector.

## Load Testing With K6

Scripts live in `k6/`.

| Command | Scenario |
| --- | --- |
| `./k6/run.sh qris` | QRIS generation load test |
| `./k6/run.sh async` | Async scan + confirm flow |

The scripts run K6 through Docker using the `grafana/k6` image.

## Phone Camera Notes

Phone camera access can fail on plain LAN HTTP because browsers often require a
secure origin for camera APIs. If the scanner does not open, check browser
permissions and try a browser/device combination that allows camera access for
your test origin.

## Extra Docs

- `report-purpose/flow.txt`
- `report-purpose/flow-mermaid.md`
- `report-purpose/changelog.md`
