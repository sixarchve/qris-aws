# QRIS Payment System

Full-stack QRIS payment simulation with a lightweight Docker runtime.

The project includes:

- Go backend API with Gin and clean repository/usecase layers
- Merchant dashboard built with React + Vite
- Customer scanner/payment app built with React + Vite
- PostgreSQL as the source of truth
- Redis cache for merchant lookup and transaction-status polling
- Merchant WebSocket notifications for successful payments

## Project Structure

```text
backend/          Go API, domain/usecase/repository code, QRIS payload logic
frontend/         Merchant dashboard, QRIS generation UI
customer-app/     Customer QR scanner and payment confirmation UI
report-purpose/   Architecture, flow, and report notes
docker-compose.yml
```

## Stack

- Go, Gin, GORM
- PostgreSQL 15
- Redis 7
- React 19 + Vite
- Nginx

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
- Payment confirmation updates PostgreSQL to `SUCCESS`, invalidates the Redis
  transaction cache, writes a local JSON receipt, and returns the existing
  confirmation response shape.
- Successful confirmations send merchant notifications directly through the
  in-process WebSocket hub at `/ws`.
- Docker Compose runs four services by default: Nginx, backend, PostgreSQL,
  and Redis.

## Environment

Create a repo-root `.env` file before running. The checked-in `.gitignore` intentionally ignores `.env`.

Example:

```env
# Database Config
DB_USER=user
DB_PASSWORD=user
DB_HOST=localhost       # Set to AWS RDS endpoint in production
DB_PORT=5432
DB_NAME=qrisdatabase
DB_SSLMODE=disable      # Set to "require" in production (for RDS)

# Redis Config
REDIS_HOST=localhost     # Set to AWS ElastiCache endpoint in production
REDIS_PORT=6379
REDIS_PASSWORD=         # Set ElastiCache AUTH password if configured
REDIS_USE_TLS=false     # Set to true for ElastiCache Transit Encryption

# AWS S3 Config (for transaction receipts)
AWS_ACCESS_KEY_ID=
AWS_SECRET_ACCESS_KEY=
AWS_REGION=ap-southeast-1
S3_BUCKET_NAME=

# WebSockets Configuration
WEBSOCKET_READ_DEADLINE=5m
WEBSOCKET_WRITE_DEADLINE=10s
WEBSOCKET_IDLE_CHECK_INTERVAL=1m
WEBSOCKET_IDLE_THRESHOLD=4m
WEBSOCKET_MAX_MESSAGE_SIZE=65536
```

> [!NOTE]
> **Host Auto-Resolution in Docker:**
> When the backend runs inside a Docker container, it automatically checks if `DB_HOST`/`REDIS_HOST` is set to `localhost` or `127.0.0.1`. If it is, it automatically resolves the hosts internally to `db` and `redis` container hostnames. This allows you to keep `localhost` in your `.env` file for both local Docker runs and host-level binary testing.

---

## Run With Docker Compose

### Local Development (including Postgres and Redis containers)
To spin up all services locally, run:
```bash
docker compose --profile local up -d
```

This starts:
- Nginx static server and reverse proxy on `http://localhost` (port 80)
  - Merchant dashboard: `http://localhost/merchant/`
  - Customer app: `http://localhost/customer/`
  - Backend API: `http://localhost/` and `/api/`
- Backend API service (on port `8080` internally)
- PostgreSQL (`qris_postgres`) exposed to the host on port `5432`
- Redis (`qris_redis`) exposed to the host on port `6379`

Useful health checks:
```bash
curl http://localhost/api/ping
curl http://localhost/api/health
curl http://localhost/api/merchants
```

### Production Deployment (AWS EC2 - bypassing local database containers)
When deploying to an EC2 instance connecting directly to RDS and ElastiCache, define your production endpoints in `.env` and start compose normally:
```bash
docker compose up -d
```
*Only Nginx and the Go backend will start. The local `db` and `redis` containers will remain off, leaving the host system completely free.*

---

## Run Apps Manually

For local development loops, start only the database and cache containers you need, then run the backend and frontend apps on your host OS:

```bash
docker compose --profile local up -d db redis
```

If the full Compose stack is already running, stop the app container you want to replace locally first, for example:

```bash
docker compose stop backend
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

The backend validates the UUID, updates the transaction to `SUCCESS`, invalidates
the Redis transaction cache, sends any WebSocket notification, and returns:

```json
{
  "data": {
    "transaction_id": "<uuid>",
    "status": "PROCESSING"
  },
  "message": "payment accepted and is being processed in background"
}
```

The response still uses `PROCESSING` for frontend compatibility, but the
lightweight runtime no longer requires a message broker container.

After a successful confirmation, the backend writes a local JSON receipt to
`RECEIPT_DIR`. Docker Compose maps this to the repo-root `./receipts` folder for
testing. The transaction response/status includes `receipt_path` after the file
is generated.

## Merchant WebSocket Notifications

The merchant dashboard connects to:

```text
GET /ws?merchant_id=<merchant_uuid>
```

When a transaction reaches `SUCCESS`, the backend sends a
`transaction_notification` message directly to every connected dashboard for
that merchant. If the merchant is disconnected, the WebSocket hub keeps a small
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

The `/api/health` endpoint checks all dependencies and returns:

```json
{
  "status": "ok",
  "timestamp": "2026-06-10T23:00:00+07:00",
  "services": {
    "postgres":  { "status": "ok" },
    "redis":     { "status": "ok" }
  }
}
```

Returns HTTP `200` when all services are healthy, `503` when any is degraded.

## Phone Camera Notes

Phone camera access can fail on plain LAN HTTP because browsers often require a
secure origin for camera APIs. If the scanner does not open, check browser
permissions and try a browser/device combination that allows camera access for
your test origin.

## Extra Docs

- `report-purpose/flow.txt`
- `report-purpose/flow-mermaid.md`
- `report-purpose/changelog.md`
