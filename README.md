# NombaSub

NombaSub is a Go-based subscription billing service for merchants building on top of Nomba. It provides tenant onboarding, customer and plan management, card checkout, direct-debit mandate creation, recurring invoice processing, payment-source storage, outbound merchant webhooks, customer email notifications, and settlement payout orchestration.

The service is designed as a backend API. It exposes REST endpoints through Gin, persists operational state in PostgreSQL with GORM, processes asynchronous jobs through RabbitMQ, integrates with Nomba for payments and transfers, and runs background schedulers for recurring billing lifecycle work.

## Table of Contents

- [Core Capabilities](#core-capabilities)
- [System Architecture](#system-architecture)
- [Domain Model](#domain-model)
- [Project Structure](#project-structure)
- [Runtime Requirements](#runtime-requirements)
- [Configuration](#configuration)
- [Getting Started](#getting-started)
- [Running with Docker](#running-with-docker)
- [Production Deployment (Coolify on EC2)](#production-deployment-coolify-on-ec2)
- [API Authentication](#api-authentication)
- [API Reference](#api-reference)
- [AI Access via MCP](#ai-access-via-mcp)
- [Billing and Subscription Lifecycle](#billing-and-subscription-lifecycle)
- [Webhooks](#webhooks)
- [Email Notifications](#email-notifications)
- [Background Jobs](#background-jobs)
- [Settlement Processing](#settlement-processing)
- [Data Persistence](#data-persistence)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Deployment Notes](#deployment-notes)
- [Operational Caveats](#operational-caveats)

## Core Capabilities

- **Tenant onboarding:** register a merchant tenant by Nomba account ID and issue a platform API key.
- **Customer management:** create, update, retrieve, and list tenant-scoped customers.
- **Plan management:** create recurring plans, version plan changes, filter plans, and optionally apply changes to existing subscriptions from the next billing cycle.
- **Subscription creation:** create subscriptions against plans and customers, with optional card token or direct-debit mandate binding.
- **Nomba checkout:** initialize card checkout orders, force tokenization, and use successful Nomba payment webhooks to store cards and activate billing.
- **Direct debit:** initialize mandates, poll pending mandate status, create bank payment sources, and debit mandates for recurring invoices.
- **Invoice automation:** create upcoming invoices, open due invoices, attempt collection, generate fallback checkout links, and pause subscriptions after uncollectible failures.
- **Retries:** optionally retry transient failed payments up to three attempts with 24-hour spacing.
- **Outbound webhooks:** persist tenant webhook deliveries, sign outgoing events, retry delivery, and record delivery attempts.
- **Email notifications:** enqueue and send customer emails for subscription, trial, invoice, payment, checkout, card-expiry, and paused-subscription events.
- **Settlements:** group pending settlement records by tenant and currency, transfer funds to tenant Nomba accounts, and emit settlement events.
- **AI-native access:** a companion MCP (Model Context Protocol) server exposes the REST API to AI clients (Claude, Cursor, Codex, ChatGPT connectors) as typed tools, resources, and reusable prompts — with API-key pass-through, per-key rate limiting, and structured error envelopes.

## System Architecture

The diagram below shows the logical service graph. For the deployed topology (Coolify, Traefik, containers on a single EC2 host), see [Production Deployment](#production-deployment-coolify-on-ec2).

```mermaid
flowchart LR
    Client["Tenant / Merchant Backend"] --> API["Gin HTTP API"]
    AI["AI Clients<br/>(Claude, Cursor, Codex, ChatGPT)"] --> MCP["MCP Server<br/>(cmd/mcp)"]
    MCP -- "X-Api-Key relay" --> API
    Nomba["Nomba APIs and Webhooks"] --> API

    API --> Services["Service Layer"]
    Services --> Repos["Repository Layer"]
    Repos --> Postgres["PostgreSQL"]

    Services --> NombaClient["Nomba Provider Client"]
    NombaClient --> Nomba

    Services --> RabbitMQ["RabbitMQ"]
    RabbitMQ --> Consumers["Queue Consumers"]
    Consumers --> Postgres
    Consumers --> TenantWebhook["Tenant Webhook URL"]
    Consumers --> SMTP["Gmail SMTP"]

    Scheduler["Cron Scheduler"] --> Services
```

The runtime starts from `cmd/api/main.go` and performs the following boot sequence:

1. Loads environment configuration.
2. Connects to PostgreSQL.
3. Runs GORM auto-migrations for all registered models.
4. Initializes the Nomba provider client.
5. Initializes repository and service containers.
6. Connects to RabbitMQ and declares queues.
7. Registers queue consumers for outbound webhooks and email delivery.
8. Registers recurring cron jobs.
9. Starts the Gin HTTP server.
10. Handles graceful shutdown on `SIGINT` and `SIGTERM`.

## Domain Model

NombaSub is organized around these main entities:

| Entity | Purpose |
| --- | --- |
| `Tenant` | A merchant account on the platform. Stores business name, Nomba account ID, API key, webhook URL, and webhook secret. |
| `Customer` | A tenant-scoped subscriber with email, optional name, phone number, external reference, and generated customer code. |
| `Plan` | A subscription product with amount, currency, interval, trial period, invoice limit, and active/inactive status. |
| `PlanVersion` | Immutable snapshots of plan terms. New versions are created when plans are updated. |
| `Subscription` | A recurring billing relationship between customer and plan version. Tracks status, billing windows, trial dates, invoice count, payment source, and retry preference. |
| `PaymentSource` | Stored card or bank mandate information for recurring charges. |
| `Invoice` | A billing-cycle receivable with status, amount due, payment attempt count, checkout link, and paid/failed timestamps. |
| `PaymentIntent` | A payment attempt against a subscription invoice or initial checkout. |
| `NombaInitiation` | Tracks initiated Nomba payment, mandate, or transfer operations and metadata needed to reconcile callbacks. |
| `WebhookDelivery` | Durable outbound webhook record for tenant event delivery. |
| `WebhookDeliveryAttempt` | Individual delivery attempt log for an outbound webhook. |
| `EmailDelivery` | Durable email job record with template context and idempotency key. |
| `Settlement` | Pending merchant settlement amount generated after successful collection. |
| `SettlementPayout` | Batched transfer of due settlements to a tenant Nomba account. |

## Project Structure

```text
cmd/api/                    HTTP entrypoint
internal/config/            Environment loading and runtime config
internal/cron/              Scheduled job registration
internal/db/                PostgreSQL connection
internal/handlers/          Gin HTTP handlers
internal/helpers/           Utility and encryption helpers
internal/mail/              SMTP mailer and HTML templates
internal/middleware/        API key authentication middleware
internal/models/            GORM models and domain enums
internal/providers/nomba/   Nomba API client, request/response types, enums
internal/queue/             RabbitMQ publisher, consumer, and job handlers
internal/repositories/      Generic GORM repository abstraction
internal/requests/          Request DTOs and validation tags
internal/responses/         Response formatting and application errors
internal/router/            HTTP route registration
internal/services/          Business logic and orchestration
```

## Runtime Requirements

- Go `1.25` or newer, matching `go.mod`.
- PostgreSQL.
- RabbitMQ.
- Nomba API credentials.
- Gmail SMTP credentials for email delivery.
- A base64-encoded 32-byte encryption key.

## Configuration

The service loads environment variables using `github.com/joho/godotenv`, so a local `.env` file is supported.

Create one from the example:

```bash
cp .env.example .env
```

The current application expects the following variables:

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `PORT` | No | `8080` | HTTP server port. |
| `DB_DSN` | Yes | None | PostgreSQL connection string. |
| `JWT_SECRET` | Yes | None | Reserved for JWT signing; currently loaded but not actively used by routes. |
| `JWT_REFRESH_SECRET` | Yes | None | Reserved for refresh-token signing; currently loaded but not actively used by routes. |
| `RABBITMQ_URL` | Yes | None | RabbitMQ AMQP URL. |
| `API_KEY_HEADER` | No | `X-Api-Key` | Header used by tenant-authenticated `/v1` routes. |
| `ENCRYPTION_KEY` | Yes | None | Base64-encoded 32-byte key for AES-256-GCM helper functions. |
| `NOMBA_CLIENT_ID` | Yes | None | Nomba OAuth client ID. |
| `NOMBA_CLIENT_SECRET` | Yes | None | Nomba OAuth client secret. |
| `NOMBA_BASE_URL` | No | `https://api.nomba.com` | Nomba API base URL. |
| `NOMBA_ACCOUNT_ID` | Yes | None | Platform Nomba account ID used in provider requests. |
| `NOMBA_SUBACCOUNT_ID` | Yes | None | Nomba sub-account ID loaded into config. |
| `NOMBA_WEBHOOK_SECRET` | Yes | None | Secret intended for validating incoming Nomba webhook signatures. |
| `NOMBA_SENDER_NAME` | No | `NombaSub Platform` | Sender name used for settlement transfers. |
| `MAILER_USER` | Yes | None | Gmail SMTP username/from address. |
| `MAILER_PASSWORD` | Yes | None | Gmail SMTP app password. |
| `MCP_PORT` | No | `8081` | HTTP port for the companion MCP server (`cmd/mcp`). |
| `NOMBASUB_ENGINE_URL` | No | `http://localhost:8080` | Base URL the MCP server uses to reach this API. |
| `MCP_RATE_LIMIT_PER_MINUTE` | No | `120` | Per-API-key request budget enforced by the MCP server. |

Generate a valid encryption key with:

```bash
openssl rand -base64 32
```

Example `.env`:

```env
PORT=8080
DB_DSN=postgres://user:password@localhost:5432/nombasub?sslmode=disable
JWT_SECRET=replace-with-long-random-secret
JWT_REFRESH_SECRET=replace-with-another-long-random-secret
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
API_KEY_HEADER=X-Api-Key
ENCRYPTION_KEY=replace-with-base64-32-byte-key
NOMBA_CLIENT_ID=replace-me
NOMBA_CLIENT_SECRET=replace-me
NOMBA_BASE_URL=https://api.nomba.com
NOMBA_ACCOUNT_ID=replace-me
NOMBA_SUBACCOUNT_ID=replace-me
NOMBA_WEBHOOK_SECRET=replace-me
NOMBA_SENDER_NAME=NombaSub Platform
MAILER_USER=merchant-mailer@example.com
MAILER_PASSWORD=replace-with-app-password
```

## Getting Started

Install dependencies:

```bash
go mod download
```

Start PostgreSQL and RabbitMQ locally. One option is Docker:

```bash
docker run --name nombasub-postgres \
  -e POSTGRES_USER=user \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=nombasub \
  -p 5432:5432 \
  -d postgres:16

docker run --name nombasub-rabbitmq \
  -p 5672:5672 \
  -p 15672:15672 \
  -d rabbitmq:3-management
```

Run the API:

```bash
make run
```

Or run directly:

```bash
go run ./cmd/api
```

For live reload with `vai`:

```bash
make dev
```

Check the service:

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{
  "data": null,
  "message": "ok",
  "status": "success"
}
```

## Running with Docker

Build the API image:

```bash
docker build -t nombasub-api .
```

Run it with your environment:

```bash
docker run --env-file .env -p 8080:8080 nombasub-api
```

The included `Dockerfile` uses a multi-stage build:

- `golang:alpine` builds the static API binary.
- `alpine:3.20` runs the final binary with CA certificates and timezone data.

## Production Deployment (Coolify on EC2)

The reference production topology is a single-node [Coolify](https://coolify.io) installation on an EC2 instance, with a Docker Compose stack (`docker-compose.yml` at the repo root) that runs the API, the MCP server, PostgreSQL, and RabbitMQ side by side. Coolify's bundled Traefik terminates TLS on `:443` and routes public traffic to the right container over the internal Docker network; Cloudflare handles DNS.

### Deployed topology

```mermaid
flowchart LR
    Merchants["Merchant backends<br/>Nomba webhooks"] -- HTTPS --> CF
    AI["AI clients<br/>(Claude, Cursor, Codex)"] -- HTTPS --> CF

    CF["Cloudflare DNS<br/>(A records, DNS-only)"] --> Traefik

    subgraph EC2["EC2 host — Coolify"]
        Traefik["Traefik<br/>:80 / :443<br/>Let's Encrypt"]
        subgraph Compose["docker-compose stack"]
            api["api container<br/>:8080"]
            mcp["mcp container<br/>:8081"]
            pg[("postgres:16<br/>postgres_data volume")]
            mq["rabbitmq:3.13<br/>rabbitmq_data volume"]
        end
        Traefik -->|SERVICE_FQDN_API| api
        Traefik -->|SERVICE_FQDN_MCP| mcp
        api --> pg
        api --> mq
        mcp -->|internal http| api
    end
```

### Infrastructure summary

| Component | Runs as | Purpose |
| --- | --- | --- |
| EC2 instance | single host | Runs Docker, Coolify, and every container in this stack. |
| Coolify | host-level service | Deploys the stack from GitHub, manages env vars, exposes a UI for logs/redeploys. |
| Traefik | container (bundled with Coolify) | Reverse proxy on `:80`/`:443`, issues and renews Let's Encrypt certs via HTTP-01, routes by `SERVICE_FQDN_*`. |
| `api` | container (`cmd/api`) | Gin HTTP server on internal `:8080`. Reached publicly via `SERVICE_FQDN_API`. |
| `mcp` | container (`cmd/mcp`) | MCP server on internal `:8081`. Reached publicly via `SERVICE_FQDN_MCP`. Talks to `api` over `http://api:8080`. |
| `postgres` | container (`postgres:16-alpine`) | Application database. State on the `postgres_data` named volume. Not exposed publicly. |
| `rabbitmq` | container (`rabbitmq:3.13-management-alpine`) | Async job queues for tenant webhooks and email delivery. State on the `rabbitmq_data` named volume. Not exposed publicly. |
| Cloudflare | external DNS | A records for the two public subdomains, DNS-only (grey cloud) so Let's Encrypt HTTP-01 works. |

### Deployment flow

1. `git push origin main`.
2. GitHub webhook triggers Coolify.
3. Coolify clones the repo into a build container, injects every env var declared in its UI as a Docker build ARG.
4. `docker compose build` runs the multi-stage `Dockerfile`, producing one image containing both `/app/api` and `/app/mcp`.
5. `docker compose up -d` recreates the stack. Traefik picks up the new container endpoints automatically; the old containers are stopped once the new ones pass their healthchecks.

## API Authentication

Public routes:

- `GET /health`
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/set-webhook-url`
- `POST /webhook/nomba`

Tenant-protected routes are grouped under `/v1` and require an API key header. By default:

```http
X-Api-Key: <tenant-api-key>
```

The API key is issued by `POST /auth/register` or returned by `POST /auth/login`.

## API Reference

All responses use this envelope:

```json
{
  "status": "success",
  "message": "Human-readable message",
  "data": {}
}
```

Errors use the same envelope with `status: "error"`.

Paginated endpoints return:

```json
{
  "data": [],
  "totalCount": 0,
  "page": 1,
  "limit": 10,
  "totalPages": 0,
  "hasNextPage": false,
  "hasPreviousPage": false
}
```

### Health

```http
GET /health
```

Returns service health.

### Register Tenant

```http
POST /auth/register
Content-Type: application/json
```

```json
{
  "businessName": "Acme Digital",
  "accountId": "nomba-account-id"
}
```

Returns:

```json
{
  "apiKey": "generated-api-key"
}
```

### Login Tenant

```http
POST /auth/login
Content-Type: application/json
```

```json
{
  "accountId": "nomba-account-id"
}
```

Returns the tenant API key for the account.

### Set Tenant Webhook URL

```http
POST /auth/set-webhook-url
Content-Type: application/json
```

```json
{
  "webhookUrl": "https://merchant.example.com/webhooks/nombasub"
}
```

This endpoint is intended to set the tenant callback URL and return a webhook secret.

### Create Customer

```http
POST /v1/customer/
X-Api-Key: <tenant-api-key>
Content-Type: application/json
```

```json
{
  "name": "Ada Lovelace",
  "email": "ada@example.com",
  "phoneNumber": "+2348012345678",
  "externalRef": "crm-customer-001"
}
```

Customer emails are unique per tenant. The service generates customer codes with a `CUST_` prefix.

### List Customers

```http
GET /v1/customer/?page=1&limit=10
X-Api-Key: <tenant-api-key>
```

Returns paginated customers with subscriptions, payment sources, and payments preloaded.

### Get Customer

```http
GET /v1/customer/{emailOrCode}
X-Api-Key: <tenant-api-key>
```

`emailOrCode` can be the customer email or generated customer code.

### Update Customer

```http
PUT /v1/customer/{emailOrCode}
X-Api-Key: <tenant-api-key>
Content-Type: application/json
```

```json
{
  "name": "Ada King",
  "phoneNumber": "+2348099999999",
  "externalRef": "crm-customer-001"
}
```

### Create Plan

```http
POST /v1/plan/
X-Api-Key: <tenant-api-key>
Content-Type: application/json
```

```json
{
  "name": "Premium Monthly",
  "description": "Monthly access to premium features",
  "amount": 500000,
  "currency": "NGN",
  "interval": "monthly",
  "intervalCount": 1,
  "trialPeriodDays": 14,
  "invoiceLimit": 12
}
```

Amounts are stored as minor units. For NGN, `500000` represents NGN 5,000.00.

Allowed intervals:

- `daily`
- `weekly`
- `bi-weekly`
- `monthly`
- `quarterly`
- `yearly`

The service creates both a `Plan` and an initial `PlanVersion`.

### List Plans

```http
GET /v1/plan/?page=1&limit=10&status=active&interval=monthly
X-Api-Key: <tenant-api-key>
```

Supported filters:

- `page`
- `limit`
- `status`: `active` or `inactive`
- `interval`
- `amount`

### Get Plan

```http
GET /v1/plan/{planCode}
X-Api-Key: <tenant-api-key>
```

### Update Plan

```http
PUT /v1/plan/{planCode}
X-Api-Key: <tenant-api-key>
Content-Type: application/json
```

```json
{
  "name": "Premium Monthly v2",
  "amount": 600000,
  "interval": "monthly",
  "intervalCount": 1,
  "trialPeriodDays": 7,
  "invoiceLimit": 12,
  "status": "active",
  "updateExistingSubscriptions": true
}
```

Each update creates a new `PlanVersion`. If `updateExistingSubscriptions` is true, existing subscriptions are marked to pick up the new plan version at the next billing cycle.

### Initialize Card Checkout

```http
POST /v1/checkout/order
X-Api-Key: <tenant-api-key>
Content-Type: application/json
```

```json
{
  "planCode": "PLN_abc12345",
  "order": {
    "callbackUrl": "https://merchant.example.com/payment/callback",
    "customerEmail": "ada@example.com",
    "orderMetaData": {
      "source": "mobile-app"
    }
  }
}
```

The service looks up the latest active plan version, gets or creates the customer by email, prevents duplicate active subscriptions for the same customer and plan, injects NombaSub metadata, sets the plan amount and currency, requests card tokenization, and creates a Nomba checkout order.

The subscription is completed when Nomba sends a `payment_success` webhook containing tokenized card data.

### Initialize Direct Debit

```http
POST /v1/checkout/direct-debit
X-Api-Key: <tenant-api-key>
Content-Type: application/json
```

```json
{
  "planCode": "PLN_abc12345",
  "customerEmail": "ada@example.com",
  "customerAccountNumber": "0123456789",
  "customerAccountName": "Ada Lovelace",
  "customerName": "Ada Lovelace",
  "customerAddress": "1 Example Street, Lagos",
  "customerPhoneNumber": "+2348012345678",
  "bankCode": "044",
  "narration": "Premium Monthly subscription",
  "frequency": "MONTHLY",
  "startDate": "2026-07-03",
  "endDate": "2027-07-03",
  "startImmediately": true,
  "orderReference": "merchant-reference-001"
}
```

Allowed mandate frequencies:

- `VARIABLE`
- `WEEKLY`
- `MONTHLY`
- `QUARTERLY`
- `EVERY_TWO_MONTHS`
- `EVERY_THREE_MONTHS`
- `EVERY_FOUR_MONTHS`
- `EVERY_FIVE_MONTHS`
- `EVERY_SIX_MONTHS`
- `EVERY_SEVEN_MONTHS`
- `EVERY_EIGHT_MONTHS`
- `EVERY_NINE_MONTHS`
- `EVERY_TEN_MONTHS`
- `EVERY_ELEVEN_MONTHS`
- `EVERY_TWELVE_MONTHS`

The service creates a pending direct-debit initiation. A background job polls Nomba every 30 minutes and creates the bank payment source and subscription when the mandate becomes active and advice has been sent.

### Create Subscription

```http
POST /v1/subscription/
X-Api-Key: <tenant-api-key>
Content-Type: application/json
```

```json
{
  "customerEmailOrCode": "ada@example.com",
  "planCode": "PLN_abc12345",
  "cardToken": "stored-card-token",
  "mandateId": null,
  "allowRetries": true
}
```

Use this endpoint when a valid payment source already exists for the customer. You may pass:

- `cardToken` to bind a stored card payment source.
- `mandateId` to bind a stored direct-debit bank payment source.
- Neither field to use the most recently created active payment source for the customer.

The service creates a subscription, sets trial/billing-cycle dates, emits `subscription.created`, and enqueues subscription emails.

### List Subscriptions

```http
GET /v1/subscription/?page=1&limit=10&customer=ada@example.com&plan=PLN_abc12345
X-Api-Key: <tenant-api-key>
```

Supported filters:

- `page`
- `limit`
- `customer`: customer ID, code, external reference, or email.
- `plan`: plan ID or code.

### Get Subscription

```http
GET /v1/subscription/{idOrCode}
X-Api-Key: <tenant-api-key>
```

Returns subscription details with customer, plan, and payment source preloaded.

### Update Subscription Mandate Status

```http
PUT /v1/subscription/{idOrCode}/mandate
X-Api-Key: <tenant-api-key>
Content-Type: application/json
```

```json
{
  "status": "SUSPENDED"
}
```

Allowed values:

- `ACTIVE`
- `SUSPENDED`
- `DELETED`

This endpoint updates the Nomba direct-debit mandate. `SUSPENDED` pauses the subscription; `DELETED` cancels the subscription and inactivates the payment source.

### Nomba Webhook Receiver

```http
POST /webhook/nomba
Content-Type: application/json
```

Consumes Nomba webhook events. Currently handled events:

- `payment_success`
- `payment_failed`
- `payment_reversal` is accepted but not implemented.
- `payout_success` is accepted but not implemented.
- `payout_failed` is accepted but not implemented.
- `payout_refund` is accepted but not implemented.

For successful online checkout subscription payments, the service stores tokenized card data, creates or updates subscriptions and invoices, creates successful payment intents, records settlements, sends customer emails, and emits tenant webhook events.

## AI Access via MCP

NombaSub ships with a companion Model Context Protocol (MCP) server at `cmd/mcp` that lets AI assistants — Claude Desktop, Cursor, Codex, ChatGPT connectors, or any MCP-aware client — query and operate the subscription engine using structured tools instead of raw HTTP calls.

The MCP server is a **thin translation layer**. It holds no business logic of its own; every tool call is forwarded to the existing REST API using the merchant's API key, so all tenant isolation, validation, and side effects continue to run through the engine's middleware and services.

### Design

- Separate binary (`cmd/mcp`) sharing the same Go module. One repo, one deploy pipeline.
- Streamable HTTP transport on `MCP_PORT` (default `8081`).
- Authentication: the AI client sends `X-Api-Key: <merchant key>`. The MCP server relays that key to the engine unchanged. Requests missing the header are rejected at the HTTP layer with `401 unauthorized` before any tool logic runs.
- Rate limiting: token-bucket, per API key. Default 120 req/min, tunable via `MCP_RATE_LIMIT_PER_MINUTE`. Exceeded budgets return `429` with `Retry-After`.
- Errors returned to tools are structured as `{code, message, retryable}` so AI clients can distinguish `invalid_input` (never retry) from `engine_error` (safe to retry).
- Health probe at `GET /health` bypasses auth for load balancers.

### Running

```bash
# terminal 1: the engine
go run ./cmd/api

# terminal 2: the MCP server
go run ./cmd/mcp
```

Both boot from the same `.env`. Point your MCP client at `http://<host>:8081/mcp` with header `X-Api-Key: <your merchant key>`.

### Tools

Fifteen tools grouped by intent. Each carries MCP annotations (`readOnlyHint`, `destructiveHint`, `idempotentHint`) so clients can gate confirmation UX correctly.

| Category | Tool | Engine endpoint | Notes |
| --- | --- | --- | --- |
| Query | `get_customers` | `GET /v1/customer/` | Filter by search, date range, pagination. |
| Query | `get_subscriptions` | `GET /v1/subscription/` | Filter by customer, plan, search. |
| Query | `get_invoices` | `GET /v1/invoice/` | Filter by status (`draft`, `open`, `paid`, `failed`, `refunded`). |
| Query | `get_plans` | `GET /v1/plan/` | Filter by status, interval, amount, date range. |
| Query | `get_payment_attempts` | `GET /v1/checkout/payment-attempts` | Per-attempt log with provider reference and failure reason. |
| Query | `get_webhook_deliveries` | `GET /v1/webhook-deliveries/` | Filter by status, event type, date. |
| Analytics | `compute_metric` | `GET /v1/dashboard/analytics` | Names: `mrr`, `arr`, `revenue`, `arpu`, `churn_rate`, etc. |
| Analytics | `compare_periods` | `GET /v1/dashboard/analytics` × 2 | Δ + % change across two windows (presets or explicit dates). |
| Analytics | `explain_metric_change` | `GET /v1/dashboard/analytics` | Current + previous + matching AI-insights + supporting signals. |
| Action | `retry_payment` | `POST /v1/invoice/:id/retry` | Destructive. Supports `dry_run: true`. |
| Action | `cancel_subscription` | `POST /v1/subscription/:id/cancel` | Destructive. Supports `dry_run: true`. |
| Action | `create_portal_link` | `POST /v1/subscription/:id/checkout-link` | Optionally emails the link. |
| Action | `send_dunning_reminder` | `POST /v1/invoice/:id/send-reminder` | Generates a link if missing; sends the checkout email. |
| Report | `generate_business_report` | analytics + orchestration | Markdown executive summary. |
| Report | `generate_dunning_report` | analytics + orchestration | Markdown recovery report with recommended tool calls. |

### Resources

Read-only URIs an AI client can browse without invoking a tool.

| URI | Contents |
| --- | --- |
| `plans://all` | Every billing plan on the merchant account. |
| `subscriptions://active` | Subscriptions currently in the active state. |
| `analytics://summary` | Full dashboard analytics response for the current billing month. |
| `events://recent` | Most recent outbound webhook deliveries. |
| `customers://{emailOrCode}` | Per-customer profile, subscriptions, and payment sources (template). |

### Prompts

Reusable prompt templates that guide an AI client through multi-step workflows using the tools above.

| Prompt | Purpose |
| --- | --- |
| `monthly_business_review` | Orchestrates `generate_business_report`, `explain_metric_change`, `compare_periods`, and (conditionally) `generate_dunning_report` into a Markdown review. Optional `focus` argument. |
| `subscription_health_check` | Walks the assistant through churn/at-risk analysis and ends with per-customer action recommendations by tool name. Optional `customer` argument. |

### Example: end-to-end from Claude Desktop

Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "nombasub": {
      "transport": {
        "type": "streamable-http",
        "url": "http://localhost:8081/mcp",
        "headers": { "X-Api-Key": "<your merchant api key>" }
      }
    }
  }
}
```

Then in a chat: "Give me this month's business review" invokes the `monthly_business_review` prompt, which in turn calls `generate_business_report`, `compare_periods`, and `explain_metric_change` — and returns a Markdown summary.

## Billing and Subscription Lifecycle

### Plan Versioning

Plans are mutable from the tenant API, but subscriptions bind to immutable `PlanVersion` records. This prevents historical billing terms from changing unexpectedly. When a plan is updated, the service creates a new version. Existing subscriptions can opt into the new version from the next billing cycle by setting `updateExistingSubscriptions` on the plan update request.

### Trials

If a plan has `trialPeriodDays > 0`:

- `trialStartDate` is set when the subscription is created.
- `trialEndDate` is calculated from the current date.
- The first billing cycle starts at the trial end date.
- A background job emits `subscription.trial_ending_soon` within three days of trial end.
- A background job sets `startedAt` and emits `subscription.billing_started` when the trial ends.

If a plan has no trial:

- Billing starts immediately.
- The current billing period is calculated from the creation date.

### Invoice Creation

The invoice service runs every three hours.

It creates draft upcoming invoices when a subscription billing cycle starts within the next three days and no draft/open invoice exists for the same due date.

When an invoice is due, the service:

1. Ensures an invoice exists.
2. Opens draft invoices.
3. Finds the active payment source.
4. Attempts card or direct-debit collection.
5. Creates a fallback checkout link if no payment source exists.
6. Marks paid invoices and advances the subscription billing cycle.
7. Marks failed/uncollectible invoices and pauses subscriptions when necessary.

### Payment Retries

If `allowRetries` is true and a payment failure is considered transient:

- The invoice remains retryable.
- `nextPaymentAttemptAt` is set 24 hours ahead.
- The subscription is marked `past_due`.
- Retries continue until the invoice has three attempts.

After retries are exhausted, or when the failure is uncollectible, the invoice is marked failed and the subscription is paused.

## Webhooks

NombaSub receives provider webhooks from Nomba and emits tenant webhooks to merchants.

### Incoming Nomba Webhooks

Incoming webhook endpoint:

```http
POST /webhook/nomba
```

The service includes Nomba signature validation support in `WebhookService.ValidateWebhookSignature`, but the HTTP handler currently has signature enforcement commented out.

### Outgoing Tenant Webhooks

Outgoing tenant webhook delivery is asynchronous:

1. A service calls `EnqueueTenantWebhook`.
2. The webhook payload is persisted to `webhook_deliveries`.
3. A RabbitMQ job is published to `send-tenant-webhook`.
4. The consumer posts the event to the tenant endpoint.
5. Each attempt is recorded in `webhook_delivery_attempts`.
6. Failed deliveries are retried up to three attempts through RabbitMQ requeue behavior.

Outgoing webhook headers:

```http
Content-Type: application/json
x-nombasub-event: invoice.paid
x-nombasub-webhook-id: <delivery-id>
x-nombasub-tenant-id: <tenant-id>
x-nombasub-timestamp: <rfc3339-utc-timestamp>
x-nombasub-signature: <base64-hmac-sha256>
```

Signature payload:

```text
eventType:webhookId:tenantId:timestamp
```

The payload is signed with the tenant webhook secret using HMAC-SHA256 and base64 encoding.

### Outgoing Event Types

Payment source events:

- `payment_method.attached`
- `payment_method.detached`
- `payment_method.updated`

Invoice and checkout events:

- `invoice.upcoming`
- `invoice.created`
- `invoice.payment_attempted`
- `invoice.paid`
- `invoice.payment_failed`
- `invoice.marked_uncollectible`
- `invoice.voided`
- `invoice.refunded`
- `checkout.created`
- `payment_success`

Subscription events:

- `subscription.created`
- `subscription.past_due`
- `subscription.paused`
- `subscription.canceled`
- `subscription.completed`
- `subscription.card_expiring`
- `subscription.trial_ending_soon`
- `subscription.billing_started`

Settlement events:

- `settlement.payout_initiated`
- `settlement.payout_failed`

Direct-debit mandate events:

- `mandate.created`
- `mandate.activated`
- `mandate.activation_failed`
- `mandate.suspended`
- `mandate.deleted`
- `mandate.debit_success`
- `mandate.debit_failed`

## Email Notifications

Email delivery is queue-backed and idempotent.

The service persists `EmailDelivery` records, publishes jobs to `send-email`, and sends HTML emails through Gmail SMTP.

Templates live in:

```text
internal/mail/templates/
```

Available templates:

- `subscription_created.html`
- `checkout_payment_required.html`
- `subscription_activated.html`
- `trial_started.html`
- `trial_ending_soon.html`
- `trial_ended_billing_started.html`
- `upcoming_invoice.html`
- `invoice_created.html`
- `payment_successful.html`
- `payment_receipt.html`
- `invoice_paid.html`
- `subscription_card_expiring.html`
- `subscription_paused.html`

Email jobs use idempotency keys such as `templateName:entityId` to prevent duplicate sends for the same event.

## Background Jobs

Cron expressions use `robfig/cron/v3` with seconds enabled.

| Job | Schedule | Purpose |
| --- | --- | --- |
| `subscription-trials` | Every 3 hours | Sends trial-ending notices and starts billing after trials end. |
| `subscription-card-expirations` | Every 3 hours | Detects cards expiring within 31 days and notifies customers/tenants. |
| `invoice-upcoming` | Every 3 hours | Creates draft invoices for billing cycles due within 3 days. |
| `invoice-processing` | Every 3 hours | Opens due invoices and attempts collection. |
| `direct-debit-mandate-poll` | Every 30 minutes | Polls pending Nomba mandates and activates subscriptions. |
| `settlement-payout` | Weekdays at 09:00 | Groups due settlements and transfers funds to tenant accounts. |

## Settlement Processing

Successful collections create pending `Settlement` records after fee deduction. The fee calculation is currently:

- `1.4%` of amount.
- Capped at `1800`.

The settlement job:

1. Loads pending settlements whose settlement date has passed.
2. Groups them by tenant and currency.
3. Creates a `SettlementPayout`.
4. Transfers funds to the tenant Nomba account.
5. Marks payout and settlement records completed or failed.
6. Emits settlement webhook events.

Settlement timestamps are currently created approximately 25 hours after successful collection.

## Data Persistence

The application uses GORM auto-migrations at startup for:

- `tenants`
- `customers`
- `plans`
- `plan_versions`
- `subscriptions`
- `invoices`
- `payment_sources`
- `payment_intents`
- `webhook_deliveries`
- `webhook_delivery_attempts`
- `nomba_webhook_events`
- `nomba_initiations`
- `email_deliveries`
- `settlement_payouts`
- `settlements`

There is no separate migrations directory in the current project. Schema changes are applied through GORM model definitions.

## Development Workflow

Common commands:

```bash
make run      # go run ./cmd/api
make dev      # vai go run ./cmd/api
make build    # go build -o bin/api ./cmd/api
make tidy     # go mod tidy
```

Format code:

```bash
gofmt -w .
```

Run all package tests:

```bash
go test ./...
```

Inspect dependency graph:

```bash
go mod graph
```

## Testing

The repository currently has no dedicated test files, but all packages should compile with:

```bash
go test ./...
```

For production readiness, recommended test coverage includes:

- Handler validation and response envelopes.
- Tenant API-key middleware.
- Plan versioning and subscription update semantics.
- Checkout initialization metadata.
- Nomba webhook reconciliation.
- Invoice retry and pause behavior.
- Direct-debit mandate polling.
- Outbound webhook signing and retry behavior.
- Settlement grouping and payout failure paths.

## Deployment Notes

Before deploying:

- Provision PostgreSQL and RabbitMQ.
- Configure all required environment variables.
- Ensure the Nomba webhook endpoint is publicly reachable at `/webhook/nomba`.
- Configure Nomba to send supported payment events to that endpoint.
- Use Gmail app passwords or replace the mailer with an organization-approved SMTP provider.
- Run at least one API instance with cron jobs enabled. If multiple replicas are deployed, introduce distributed locking or isolate workers to avoid duplicate cron processing.
- Ensure the runtime can reach Nomba APIs, tenant webhook URLs, RabbitMQ, PostgreSQL, and SMTP.
- Monitor queue depth, webhook delivery failure rates, email failures, settlement failures, and Nomba provider errors.

## Operational Caveats

These notes reflect the current implementation and are useful for maintainers planning hardening work:

- `.env.example` is missing some variables that `config.Load()` requires. Use the complete configuration table above when setting up the application.
- `POST /auth/set-webhook-url` reads tenant context, but the route is currently outside the `/v1` API-key middleware group. As written, no tenant ID is set for that handler by the router.
- `SetWebhookUrl` generates and returns a webhook secret, but does not currently persist it to `tenant.WebhookSecret`. Outgoing webhooks require a stored webhook URL and secret before they are enqueued.
- Incoming Nomba webhook signature validation exists in the service layer, but enforcement is commented out in the webhook handler.
- Some accepted Nomba webhook event types are stubbed and do not yet mutate local state.
- The RabbitMQ reconnect loop reconnects the underlying connection/channel, but existing consumers are not explicitly re-registered after reconnection.
- Cron jobs run in-process. Multiple running API instances can execute the same cron work unless external locking or worker separation is added.
- GORM auto-migration is convenient for development, but organizations commonly replace or supplement it with explicit, reviewed database migrations in production.

