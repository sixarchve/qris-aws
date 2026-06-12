# QRIS Payment System Flow

```mermaid
flowchart TD
    A[Backend Start] --> B[Load repo-root .env or container env]
    B --> C[Connect PostgreSQL with Retry Loop]
    C --> D[Create pgcrypto extension]
    D --> E[AutoMigrate merchants and transactions]
    E --> F[Seed default merchants]
    F --> G[Connect Redis with TLS/Auth]
    G --> H[Warm merchant cache]
    H --> WH[Start WebSocket hub]
    WH --> K[Start Gin HTTP server on 8080]
    K --> L[Start Nginx Proxy on 80]

    MD[Merchant Dashboard] --> ML[GET /api/merchants]
    ML --> MP[Query active merchants from PostgreSQL]
    MP --> MR[Return merchant UUIDs and QRIDs]

    MD --> Q1[GET /api/qris]
    Q1 --> Q2[Validate merchant UUID and amount]
    Q2 --> Q3[Load merchant from PostgreSQL]
    Q3 --> Q4[Cache merchant in Redis]
    Q4 --> Q5[Prefetch related merchants]
    Q5 --> Q6[Generate QRIS payload with CRC]
    Q6 --> Q7[Return qris_payload]

    CA[Customer App] --> S1[Scan QRIS payload]
    S1 --> S2[Extract QRID tag 26.01 and amount tag 54]
    S2 --> S3[POST /api/transactions/scan]
    S3 --> S4[Find merchant by UUID or QRID]
    S4 --> S5{Merchant in Redis?}
    S5 -->|Yes| S6[Use cached merchant]
    S5 -->|No| S7[Query PostgreSQL and cache merchant]
    S6 --> S8[Validate QR CRC, merchant, amount]
    S7 --> S8
    S8 --> S9[Create PENDING transaction in PostgreSQL]
    S9 --> S10[Cache transaction in Redis]
    S10 --> S11[Return transaction_id]

    CA --> ST1[GET /api/transactions/:id]
    ST1 --> ST2{Transaction in Redis?}
    ST2 -->|Hit| ST3[Return cached transaction]
    ST2 -->|Miss or corrupt| ST4[Query PostgreSQL]
    ST4 --> ST5[Cache fresh transaction]
    ST5 --> ST6[Return DB transaction]

    CA --> AC1[POST /api/transactions/:id/confirm]
    AC1 --> AC2[Update PostgreSQL status to SUCCESS]
    AC2 --> AC3[Delete Redis transaction cache]
    AC3 --> AC4[Save Receipt to S3 and/or local storage]
    AC4 --> AC5[Trigger asynchronous WebSocket broadcast]
    AC5 --> WS[Push transaction_notification over /ws]
    WS --> MD

    MD --> WSC[GET /ws?merchant_id]
    WSC --> WH
    MD --> WSS[GET /api/ws/status]
    WSS --> WH

    User[Browser / Client] -->|Port 80| NginxRouter[Nginx Reverse Proxy]
    NginxRouter -->|/merchant/| MD
    NginxRouter -->|/customer/| CA
    NginxRouter -->|/api/| API[Go Backend API]
    NginxRouter -->|/ws| WH
```

## Notes

- **PostgreSQL** is the source of truth for all merchants and transactions.
- **Redis** caches active merchants and recent transactions.
- **S3 Receipt Store** saves transaction receipts securely to AWS S3. A local store serves as a backup or when S3 credentials are not supplied.
- **WebSockets** stream successful payment notifications to the merchant dashboard. `/api/ws/status` exposes websocket statistics.
- **Nginx** handles reverse proxying. Its dependency on the backend is set to `service_started`, enabling Nginx to start immediately and expose the port `/api/health` even if the backend is booting or degraded.
