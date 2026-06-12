# QRIS Payment System - AWS Deployment Architecture

This document details the production AWS cloud architecture, network topology, data flow, and infrastructure configurations for deploying the QRIS Payment System in AWS.

---

## 1. AWS Architecture Topology

In production, the application utilizes a secure AWS VPC network layout that isolates stateless application servers from managed database, caching, and storage systems:

```
+-----------------------------------------------------------------------------------+
|                                     AWS VPC                                       |
|                                                                                   |
|   +---------------------------------------------------------------------------+   |
|   |                         Public Subnets (DMZ)                              |   |
|   |                                                                           |   |
|   |   +--------------------+               +------------------------------+   |   |
|   |   |   Internet Gateway |               |      EC2 Instance (Docker)   |   |   |
|   |   +---------+----------+               |                              |   |   |
|   |             |                          |   +-----------------------+  |   |   |
|   |             | (Port 80/443)            |   |         Nginx         |  |   |   |
|   |             v                          |   |     (Docker Proxy)    |  |   |   |
|   |     [ALB / Client] ------------------> |   +-----------+-----------+  |   |   |
|   |                                        |               | (Port 8080)  |   |   |
|   |                                        |   +-----------v-----------+  |   |   |
|   |                                        |   |      Go Backend       |  |   |   |
|   |                                        |   |    (Docker Server)    |  |   |   |
|   |                                        |   +-----------+-----------+  |   |   |
|   +--------------------------------------------------------|------------------+   |
|                                                            |                      |
|   +--------------------------------------------------------|------------------+   |
|   |                         Private Subnets                |                      |
|   |                                                        | (VPC Routing)        |
|   |                                                        v                      |
|   |     +-------------------------+            +--------------------------+       |
|   |     |    AWS RDS Postgres     |            |  AWS ElastiCache Redis   |       |
|   |     |       (Port 5432)       |            |       (Port 6379)        |       |
|   |     +-------------------------+            +--------------------------+       |
|   +---------------------------------------------------------------------------+   |
+------------------------------------------------------------|----------------------+
                                                             |
                                                             v (HTTPS / IAM Role)
                                                 +--------------------------+
                                                 |        Amazon S3         |
                                                 |     (Receipts Bucket)    |
                                                 +--------------------------+
```

---

## 2. Infrastructure Components

### 1. Application Layer (Amazon EC2 Instance)
* Runs **Docker Engine** to host the stateless application containers:
  * **`qris_nginx`:** Listens on Port 80, serves React static builds, and reverse-proxies `/api` and `/ws` requests.
  * **`qris_backend`:** Runs the Go Gin API server.
* Communicates internally using a bridge Docker network namespace.
* Evaluates configuration inputs directly from variables set in the EC2 host `.env` file.

### 2. Database Layer (Amazon RDS PostgreSQL)
* **Engine:** PostgreSQL 15.
* **Security:** Configured with **Publicly Accessible = No** and isolated inside private database subnets.
* **SSL:** Enforced via `DB_SSLMODE=require` inside the Go database config to secure all connections from the EC2 instance.
* **Auto-Schema:** The Go backend executes GORM schema migrations (`AutoMigrate`) and seeds default merchant tables automatically on startup.

### 3. Caching Layer (Amazon ElastiCache Redis)
* **Security:** Runs inside a private subnet and requires **Redis AUTH** password verification.
* **Transit Encryption:** Enabled (TLS/SSL) to encrypt all data in transit using the `REDIS_USE_TLS=true` configuration.
* **Eviction Policy:** Configured with `allkeys-lru` and a strict memory ceiling (e.g. 64MB) to serve as a fast ephemeral cache layer.

### 4. Storage Layer (Amazon S3)
* **Bucket:** Private S3 bucket utilizing programmatic access keys.
* **Policy Constraints:** Restricted using an IAM policy allowing `s3:PutObject` and `s3:GetObject` only on the `/receipts/*` prefix.
* **Composite Store:** If S3 cannot be reached, the application automatically triggers a local directory storage callback for redundancy.

---

## 3. Network Security & Port Rules

To prevent unauthorized access, security groups are strictly isolated using the following inbound/outbound rules:

| Source Security Group | Target Security Group | Protocol | Port | Description |
| :--- | :--- | :--- | :--- | :--- |
| Any (0.0.0.0/0) | **EC2 Security Group** | TCP | 80 / 443 | Web access to Nginx proxy |
| Admin IP / VPN | **EC2 Security Group** | TCP | 22 | Secure SSH access |
| EC2 Security Group | **RDS Security Group** | TCP | 5432 | Go database connection to Postgres |
| EC2 Security Group | **ElastiCache Security Group** | TCP | 6379 | Go cache connection to Redis |
| EC2 Security Group | **AWS IAM Role / S3** | HTTPS | 443 | Outbound uploads of JSON receipt files |

---

## 4. AWS Integration Configuration (.env)

The application pulls configuration directly from the environment. On your EC2 server, populate `.env` with:

```env
# AWS RDS Connection
DB_USER=your_rds_username
DB_PASSWORD=your_rds_password
DB_HOST=your-rds-endpoint.xxxxxx.ap-southeast-1.rds.amazonaws.com
DB_PORT=5432
DB_NAME=qrisdatabase
DB_SSLMODE=require

# AWS ElastiCache Connection
REDIS_HOST=your-elasticache-endpoint.xxxxxx.cache.amazonaws.com
REDIS_PORT=6379
REDIS_PASSWORD=your_elasticache_auth_token
REDIS_USE_TLS=true

# AWS S3 Receipts Storage
AWS_ACCESS_KEY_ID=YOUR_IAM_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY=YOUR_IAM_SECRET_ACCESS_KEY
AWS_SESSION_TOKEN=YOUR_AWS_SESSION_TOKEN    # Required only for AWS Academy Learner Labs
AWS_REGION=ap-southeast-1
S3_BUCKET_NAME=your-s3-bucket-name
```
