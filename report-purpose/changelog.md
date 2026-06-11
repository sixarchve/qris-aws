Change Report Compared With Upstream Main
=========================================

Comparison base: upstream/main
Current branch: newMonitor

This document summarizes the current branch delta against upstream/main. The
comparison was refreshed with `git fetch upstream main` before this report was
updated.


1. Executive Summary
--------------------

The branch changes the project from a simpler QRIS backend and local monitoring
prototype into a fuller QRIS payment system:

- Backend code is reorganized into cleaner handler, middleware, usecase,
  repository, domain, worker, and internal packages.
- Docker Compose is moved to the repository root and now runs the API,
  PostgreSQL, Redis, RabbitMQ, frontend apps, Prometheus, Grafana, and Nginx.
- Nginx acts as a unified reverse proxy on port 80 routing requests to the frontends (/merchant/, /customer/) and the backend API (/api, /).
- QRIS generation and scan validation are backed by PostgreSQL merchant data,
  QR CRC validation, Redis cache-aside reads, and explicit transaction status
  APIs.
- Payment confirmation uses an async path through RabbitMQ: `/api/transactions/:id/confirm`
- Merchant dashboards receive successful payment notifications over WebSocket.
- Prometheus, Grafana, and K6 replace the removed static monitoring
  pages and old `tests_script/` load-test files.


2. Backend Changes
------------------

Added or changed:

- `backend/cmd/main.go` now wires config loading, PostgreSQL, Redis, RabbitMQ,
  usecases, handlers, workers, WebSocket hub, and graceful shutdown.
- `backend/config/config.go` centralizes environment configuration.
- `backend/delivery/handler/` now has dedicated merchant, QRIS, transaction,
  ping, and router files.
- `backend/delivery/middleware/prometheus.go` adds Prometheus collectors and
  request instrumentation.
- `backend/domain/entity/` and `backend/domain/repository/` replace the older
  model/service coupling with explicit entities and repository interfaces.
- `backend/repository/postgres/` contains merchant and transaction repository
  implementations.
- `backend/repository/redis/` contains merchant cache, merchant prefetch, and
  transaction cache behavior.
- `backend/repository/rabbitmq/rabbitmq.go` declares both
  `payment_confirmations` and `merchant_notifications` queues.
- `backend/internal/qris/` owns QRIS payload generation/parsing and CRC tests.
- `backend/internal/websocket/` adds the merchant WebSocket hub and connection
  handler.
- `backend/worker/payment_consumer.go` processes async payment confirmations
  and merchant notification deliveries.

Removed or replaced:

- `backend/delivery/handler/rest.go`
- `backend/repository/database/loadenv.go`
- `backend/repository/database/pg.go`
- old service files under `backend/usecase/service/`
- old customer transaction usecase under `backend/usecase/customer/`
- static monitoring pages under `backend/monitoring/`


3. API And Runtime Behavior
---------------------------

Current main routes:

- `GET /api/ping`
- `GET /api/merchants`
- `GET /api/qris?merchant_id=<merchant_uuid>&amount=<amount>`
- `GET /api/transactions/:id`
- `GET /api/ws/status?merchant_id=<merchant_uuid>`
- `GET /ws?merchant_id=<merchant_uuid>`
- `GET /metrics`
- `POST /api/transactions/scan`
- `POST /api/transactions/:id/confirm`

Behavior added on this branch:

- QRIS scan accepts a merchant UUID or QRID, validates the QR payload, checks
  merchant and amount consistency, creates a PENDING transaction, and caches it.
- Transaction status reads Redis first and falls back to PostgreSQL.
- Async confirmation queues the transaction and returns PROCESSING quickly.
- Confirmation publishes a merchant notification after SUCCESS.
- WebSocket clients receive `transaction_notification` events by merchant UUID.


4. Observability And Testing
----------------------------

Added:

- root `prometheus.yml`
- Grafana datasource and dashboard provisioning under `grafana/provisioning/`
- Grafana dashboard JSON under `grafana/dashboards/golang-metrics.json`
- Prometheus metrics for:
  - HTTP request totals and durations
  - transaction creation
  - payment worker processing counts and durations
  - cache lookup and write counts
- K6 scripts under `k6/`:
  - `qris_generation.js`
  - `scan_async_payment.js`
  - `run.sh`

Removed:

- old `tests_script/` JavaScript files
- old dashboard JSON under `tests_script/`
- static backend monitoring HTML pages


5. Frontend And Customer App Changes
------------------------------------

Merchant dashboard:

- Loads merchants from the backend and uses merchant UUIDs for QRIS generation.
- Generates QRIS payloads by selected merchant and submitted amount.
- Opens a merchant-scoped WebSocket connection.
- Displays live payment notifications when transactions reach SUCCESS.

Customer app:

- Extracts merchant QRID and amount from scanned QRIS payloads.
- Creates transactions through `/api/transactions/scan`.
- Confirms payment through the async endpoint.
- Polls transaction status.


6. Environment And Operations
-----------------------------

Added root-level `.env_example` and root-level `docker-compose.yml`.

Important environment groups:

- PostgreSQL: `DB_USER`, `DB_PASSWORD`, `DB_HOST`, `DB_PORT`, `DB_NAME`
- Redis: `REDIS_HOST`, `REDIS_PORT`
- RabbitMQ: `RABBITMQ_USER`, `RABBITMQ_PASSWORD`, `RABBITMQ_HOST`,
  `RABBITMQ_PORT`
- WebSocket tuning: `WEBSOCKET_READ_DEADLINE`,
  `WEBSOCKET_WRITE_DEADLINE`, `WEBSOCKET_IDLE_CHECK_INTERVAL`,
  `WEBSOCKET_IDLE_THRESHOLD`, `WEBSOCKET_MAX_MESSAGE_SIZE`
- Grafana credentials


7. Test Coverage Added
----------------------

Added or updated backend tests include:

- QRIS payload tests under `backend/internal/qris/`
- QRIS usecase tests
- transaction usecase tests
- payment consumer tests


8. Net File-Level Delta
-----------------------

The refreshed diff against upstream/main shows 81 changed paths, including:

- 3,848 insertions
- 4,787 deletions
- root docs and Compose additions
- monitoring/test stack additions
- backend architecture reorganization
- removal of old monitoring and test-script artifacts
