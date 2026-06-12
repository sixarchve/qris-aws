Change Report Compared With Upstream Main
=========================================

Comparison base: upstream/main
Current branch: main

This document summarizes the current branch delta against upstream/main.


1. Executive Summary
--------------------

The branch changes the project from a simpler QRIS backend and local monitoring prototype into a fuller, cloud-ready QRIS payment system:

- Backend code is reorganized into cleaner handler, middleware, usecase, repository, domain, and internal packages.
- Integrated AWS RDS (PostgreSQL) with configurable SSL/TLS support (`DB_SSLMODE`).
- Integrated AWS ElastiCache (Redis) with AUTH password and transit encryption/TLS support.
- Integrated AWS S3 receipt store for transaction receipts, uploading generated receipts directly to S3 and falling back to local file storage if S3 is unconfigured.
- Docker Compose is consolidated into a single configuration utilizing Docker Compose Profiles (`local` profile for local db and redis containers).
- Nginx acts as a unified reverse proxy on port 80 routing requests to the frontends (/merchant/, /customer/) and the backend API (/api, /ws, /).
- Nginx's dependency condition is relaxed to `service_started` so that it boots immediately during backend errors or startup lag, allowing developers to always reach health checks.
- QRIS generation and scan validation are backed by PostgreSQL merchant data, QR CRC validation, Redis cache-aside reads, and explicit transaction status APIs.
- Payment confirmation and status checks utilize a robust PostgreSQL transaction write path and Redis caching.
- Merchant dashboards receive successful payment notifications over an in-process WebSocket connection immediately upon payment confirmation.


2. Backend Changes
------------------

Added or changed:

- `backend/cmd/main.go` wires config loading, PostgreSQL, Redis, S3 receipt store, usecases, handlers, WebSocket hub, and graceful shutdown.
- `backend/config/config.go` centralizes configuration, auto-detects if running inside a Docker container, and falls back from `localhost`/`127.0.0.1` to internal Compose hostnames (`db` / `redis`) dynamically.
- `backend/delivery/handler/` has dedicated merchant, QRIS, transaction, ping, and router files.
- `backend/domain/entity/` and `backend/domain/repository/` replace older model/service coupling with explicit entities and repository interfaces.
- `backend/repository/postgres/` contains merchant and transaction repository implementations with a database connection retry loop (5 attempts, 2s sleep) to withstand concurrent Docker startup lag.
- `backend/repository/redis/` contains merchant cache, merchant prefetch, and transaction cache behavior.
- `backend/repository/s3/` implements S3ReceiptStore for uploading JSON receipt files directly to an AWS S3 bucket.
- `backend/repository/local/` implements CompositeReceiptStore and LocalReceiptStore to handle local and AWS S3 multi-store backups.
- `backend/internal/qris/` owns QRIS payload generation/parsing and CRC validation.
- `backend/internal/websocket/` adds the merchant WebSocket hub and connection handler.

Removed or replaced:

- `backend/delivery/handler/rest.go`
- `backend/repository/database/loadenv.go`
- `backend/repository/database/pg.go`
- old service files under `backend/usecase/service/`
- old customer transaction usecase under `backend/usecase/customer/`


3. API And Runtime Behavior
---------------------------

Current main routes:

- `GET /api/ping`
- `GET /api/health`
- `GET /api/merchants`
- `GET /api/qris?merchant_id=<merchant_uuid>&amount=<amount>`
- `GET /api/transactions/:id`
- `GET /api/ws/status?merchant_id=<merchant_uuid>`
- `GET /ws?merchant_id=<merchant_uuid>`
- `POST /api/transactions/scan`
- `POST /api/transactions/:id/confirm`

Behavior added on this branch:

- QRIS scan accepts a merchant UUID or QRID, validates the QR payload, checks merchant and amount consistency, creates a PENDING transaction, and caches it.
- Transaction status reads Redis first and falls back to PostgreSQL.
- Payment confirmation updates status to SUCCESS in the database, invalidates the cache, saves the transaction receipt to S3/local storage, and fires an async goroutine to stream notifications.
- WebSocket clients receive `transaction_notification` events by merchant UUID immediately upon payment.


4. Frontend And Customer App Changes
------------------------------------

Merchant dashboard:

- Loads merchants from the backend and uses merchant UUIDs for QRIS generation.
- Generates QRIS payloads by selected merchant and submitted amount.
- Opens a merchant-scoped WebSocket connection.
- Displays live payment notifications when transactions reach SUCCESS.

Customer app:

- Extracts merchant QRID and amount from scanned QRIS payloads.
- Creates transactions through `/api/transactions/scan`.
- Confirms payment through the API.
- Polls transaction status.


5. Environment And Operations
-----------------------------

Added root-level `.env_example` and root-level `docker-compose.yml`.

Important environment groups:

- PostgreSQL: `DB_USER`, `DB_PASSWORD`, `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_SSLMODE`
- Redis: `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_USE_TLS`
- AWS S3: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`, `S3_BUCKET_NAME`
- WebSocket tuning: `WEBSOCKET_READ_DEADLINE`, `WEBSOCKET_WRITE_DEADLINE`, `WEBSOCKET_IDLE_CHECK_INTERVAL`, `WEBSOCKET_IDLE_THRESHOLD`, `WEBSOCKET_MAX_MESSAGE_SIZE`


6. Test Coverage Added
----------------------

Added or updated backend tests include:
- QRIS payload tests under `backend/internal/qris/`
- Composite Receipt Store tests under `backend/repository/local/`
- QRIS usecase tests
- Transaction usecase tests
