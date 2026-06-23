# ION Insurance Test App

A full-stack **Beckn v2 motor insurance** demo built on the ION protocol. The application implements both sides of the Beckn network:

- **BPP (Insurer-facing)** — insurers publish products, manage policies, view claims, ratings, and support tickets
- **BAP (Customer-facing)** — customers discover products, buy policies, track status, rate their experience, and raise support tickets

The stack uses **beckn-onix** as the protocol adapter (handling signing, routing, schema validation) and a Central Server (CS) mock to simulate the Beckn registry.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        BAP Side                              │
│  bap-frontend (React)  →  bap-application (Go/Gin, :8083)  │
│                              ↕ Redis pub/sub                 │
│                         onix-bap (:8081)                     │
└───────────────────────────────┬─────────────────────────────┘
                                │  Beckn v2 protocol
┌───────────────────────────────▼─────────────────────────────┐
│                        BPP Side                              │
│  bpp-frontend (React)  →  bpp-application (Go/Gin, :8080)  │
│                         onix-bpp (:8082)                     │
└─────────────────────────────────────────────────────────────┘
                PostgreSQL (:5435)  |  Redis (:6380)
                      CS Mock (:9090)
```

| Service | URL | Description |
|---|---|---|
| BAP Frontend | http://localhost:3000 | Customer portal — discover, buy, track policies |
| BPP Frontend | http://localhost:3001 | Insurer admin — dashboard, publish, policies, claims |
| BPP API | http://localhost:8080 | Insurer backend (Go/Gin) |
| BAP API | http://localhost:8083 | Customer backend (Go/Gin) |
| onix-bap | http://localhost:8081 | Beckn protocol adapter (BAP side) |
| onix-bpp | http://localhost:8082 | Beckn protocol adapter (BPP side) |
| CS Mock | http://localhost:9090 | Central Server / registry mock |
| PostgreSQL | localhost:5435 | Two DBs: `insurance_bpp` + `insurance_bap` |
| Redis | localhost:6380 | BAP async callback queue |

---

## Prerequisites

- **Docker** ≥ 24 with **Docker Compose** v2
- **`gh` CLI** (optional, for repo management)
- No local Go or Node installs required — everything builds inside Docker

---

## Quick Start

### 1. Clone and start

```bash
git clone https://github.com/remiges-Tushar/ion-insurance-testapp.git
cd ion-insurance-testapp
docker compose up --build -d
```

First build takes ~3–5 minutes (Go compilation + npm install). Subsequent starts are fast.

### 2. Seed the catalog

Wait ~10 seconds for all services to be healthy, then:

```bash
cd catalog-seed
chmod +x publish-catalog.sh
./publish-catalog.sh
```

This script:
1. Registers insurer **PT Asuransi Maju** with OJK license `OJK-KEP-234`
2. Creates two insurance products: `MOTOR_COMPREHENSIVE` and `MOTOR_THIRD_PARTY` (Zone 3)
3. Creates premium offers for each product
4. Publishes the catalog to the Beckn network

Default credentials after seeding:
- **BPP login**: `admin@ptasuransimaju.co.id` / `SecurePass123`

### 3. Open the UIs

| App | URL | Notes |
|---|---|---|
| Customer Portal (BAP) | http://localhost:3000 | No login required |
| Insurer Dashboard (BPP) | http://localhost:3001 | Login with seeded credentials |

---

## End-to-End Demo Flow

### BPP side (Insurer)
1. Open http://localhost:3001 → log in
2. **Publish** tab → seed script already did this, or publish manually
3. **Inventory → By Catalog** — see published products and offers in tree view
4. **Overview** — watch stats update as customers buy policies

### BAP side (Customer)
1. Open http://localhost:3000
2. Click **Browse Products** → search returns MOTOR_COMPREHENSIVE and MOTOR_THIRD_PARTY cards
3. Click **Get Quote** on any product
4. **Vehicle Details** — pre-filled with demo vehicle (Toyota Avanza, IDV 210,000,000)
5. **Quote** — see OJK-calculated premium with breakup
6. **KYC** — pre-filled with demo policyholder (Budi Santoso, NIK 3171012345678901)
7. **Payment** — click "I Have Paid" to confirm
8. **Policy Issued** — policy number, certificate URL, coverage dates
9. Rate the insurer (1–5 stars + comment)
10. Raise a support ticket

### BPP side — after purchase
- **Policies** tab shows the new `ACTIVE` policy
- **Ratings & Reviews** shows the star rating with score distribution
- **Support Tickets** shows the submitted ticket linked to the policy

---

## Project Structure

```
ion-insurance-testapp/
├── docker-compose.yml             # All services wired together
├── catalog-seed/
│   ├── publish-catalog.sh         # One-shot seed script
│   └── motor-insurance-catalog.json
├── config/                        # beckn-onix routing configs
│   ├── insurance-bap.yaml
│   ├── insurance-bpp.yaml
│   ├── routing-bap-caller.yaml
│   ├── routing-bap-receiver.yaml
│   ├── routing-bpp-caller.yaml
│   └── routing-bpp-receiver.yaml
├── cs-mock/                       # Central Server mock (Go)
│   ├── main.go
│   └── Dockerfile
├── postgres/
│   └── init-dbs.sh               # Creates insurance_bpp + insurance_bap databases
├── bpp-application/               # Insurer backend (Go + Gin)
│   ├── cmd/server/
│   ├── internal/
│   │   ├── api/                   # Request/response types
│   │   ├── db/migrations/         # SQL migrations (tern)
│   │   │   ├── 001_auth.sql
│   │   │   ├── 002_catalogs.sql
│   │   │   ├── 003_inventory.sql
│   │   │   ├── 004_policies.sql
│   │   │   ├── 005_audit.sql
│   │   │   ├── 006_support.sql
│   │   │   └── 007_providers_v2.sql
│   │   ├── http/                  # Gin handlers + routes
│   │   ├── repository/
│   │   └── service/
│   │       ├── auth.go            # JWT auth
│   │       ├── beckn.go           # All Beckn webhook handlers
│   │       └── catalog.go         # Catalog publish + sanitisation
│   ├── Dockerfile
│   └── Dockerfile.migrate
├── bpp-frontend/                  # Insurer dashboard (React 19 + Tailwind 4)
│   └── src/pages/
│       ├── LoginPage.jsx
│       ├── RegisterPage.jsx
│       ├── OverviewPage.jsx       # Stats + conversion funnel
│       ├── PoliciesPage.jsx       # Issued policies list
│       ├── PolicyDetailPage.jsx
│       ├── InventoryPage.jsx      # Products / Offers / By Catalog tabs
│       ├── PublishPage.jsx        # Step-through catalog publish form
│       ├── CatalogsPage.jsx
│       ├── ClaimsPage.jsx
│       ├── MessagesPage.jsx       # Beckn audit log
│       ├── RatingsPage.jsx        # Star ratings with distribution chart
│       └── SupportPage.jsx        # Support tickets with status breakdown
├── bap-application/               # Customer backend (Go + Gin)
│   ├── cmd/server/
│   ├── internal/
│   │   ├── db/migrations/
│   │   │   ├── 001_transactions.sql
│   │   │   ├── 002_snapshots.sql
│   │   │   └── 003_audit.sql
│   │   ├── http/
│   │   └── service/
│   │       ├── client.go          # select/init/confirm/rate/support flows
│   │       └── callback.go        # Redis async pub/sub bridge
│   ├── Dockerfile
│   └── Dockerfile.migrate
└── bap-frontend/                  # Customer portal (React 19 + Tailwind 4)
    └── src/pages/
        ├── HeroPage.jsx           # Product discovery grid
        ├── PolicyFlowPage.jsx     # 5-step purchase wizard
        └── PolicyHistoryPage.jsx  # My Policies with coverage progress bar
```

---

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.22 + Gin |
| Frontend | React 19 + Vite + Tailwind CSS 4 |
| Animations | Framer Motion |
| Icons | Lucide React |
| i18n | i18next + react-i18next (English / Bahasa Indonesia) |
| Database | PostgreSQL 16 |
| Cache / Queue | Redis (BAP async callback) |
| Protocol | Beckn v2 via beckn-onix (fidedocker/onix-adapter) |
| DB Migrations | pgx-tern |
| Auth | JWT (BPP only) |
| Container | Docker + Docker Compose v2 |

---

## BPP Database Schema (`insurance_bpp`)

```
provider_accounts   — insurer login (company, OJK license, JWT)
catalogs            — published catalogs
catalog_publish_results — CDS publish callbacks
providers           — provider metadata (locations JSONB)
resources           — InsuranceProduct definitions
offers              — PolicyQuote offers (zone, premium rates)
policies            — issued policies (PENDING_ISSUANCE → ACTIVE)
payments            — payment records per policy
claims              — incoming claims
beckn_message_log   — full Beckn protocol audit log
support_tickets     — customer support requests
ratings             — policyholder star ratings + feedback
```

## BAP Database Schema (`insurance_bap`)

```
transactions        — transaction state machine
contract_snapshots  — on_confirm / on_select / on_init raw payloads
beckn_message_log   — full Beckn protocol audit log
```

---

## API Reference

### BPP API (`:8080`)

| Method | Path | Description |
|---|---|---|
| POST | `/api/v1/auth/register` | Register new insurer |
| POST | `/api/v1/auth/login` | Login → JWT |
| GET | `/api/v1/dashboard/stats` | Overview stats |
| GET | `/api/v1/policies` | All issued policies |
| GET | `/api/v1/inventory/resources` | Published products |
| GET | `/api/v1/inventory/offers` | Published offers |
| GET | `/api/v1/catalogs` | All catalogs |
| POST | `/api/v1/catalog/publish` | Publish catalog to Beckn |
| GET | `/api/v1/ratings` | Customer ratings |
| GET | `/api/v1/support-tickets` | Support tickets |
| GET | `/api/v1/messages` | Beckn audit log |
| POST | `/webhook/search` | Beckn search handler |
| POST | `/webhook/select` | Beckn select → OJK premium calc |
| POST | `/webhook/init` | Beckn init → KYC validation |
| POST | `/webhook/confirm` | Beckn confirm → issue policy |
| POST | `/webhook/rate` | Beckn rate → save rating |
| POST | `/webhook/support` | Beckn support → save ticket |

### BAP API (`:8083`)

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/discover` | Search for insurance products |
| POST | `/api/v1/select` | Select product + vehicle details |
| POST | `/api/v1/init` | Submit KYC |
| POST | `/api/v1/confirm` | Confirm payment |
| POST | `/api/v1/rate` | Submit rating (score 1–5 + feedback) |
| POST | `/api/v1/support` | Raise support ticket |
| GET | `/api/v1/policies` | Customer's policy history |
| POST | `/webhook/on_select` | Beckn on_select callback |
| POST | `/webhook/on_init` | Beckn on_init callback |
| POST | `/webhook/on_confirm` | Beckn on_confirm callback |
| POST | `/webhook/on_rate` | Beckn on_rate callback |
| POST | `/webhook/on_support` | Beckn on_support callback |

---

## Configuration

All environment variables are set in `docker-compose.yml`. Key overrides:

| Variable | Default | Description |
|---|---|---|
| `BPP_JWT_SECRET` | `insurance-bpp-jwt-secret-change-in-production` | JWT signing key — change in production |
| `BPP_DB_*` | see compose | BPP PostgreSQL connection |
| `BAP_DB_*` | see compose | BAP PostgreSQL connection |
| `BAP_REDIS_ADDR` | `redis:6379` | Redis for async callbacks |

For custom onix routing, edit the YAML files in `config/`.

---

## Development

To rebuild and restart a single service after code changes:

```bash
# Rebuild + restart BPP backend
docker compose build bpp && docker compose up -d bpp

# Rebuild + restart BAP frontend
docker compose build bap-frontend && docker compose up -d bap-frontend

# Tail logs
docker compose logs -f bpp
docker compose logs -f bap
```

To reset the databases:

```bash
docker compose down -v          # removes postgres volume
docker compose up --build -d    # rebuilds from scratch
./catalog-seed/publish-catalog.sh
```

---

## Key Design Decisions

- **Async → sync bridge**: Beckn callbacks are async. The BAP uses Redis pub/sub (`forwardAndWait`) to block the HTTP response until the `on_*` callback arrives (30 s timeout), so the React frontend sees a synchronous flow.
- **AnimatePresence guard**: Framer Motion's exit animations keep unmounting components alive in the DOM. All submit handlers use a `useRef` guard (`submitting.current`) that is NOT reset on success, preventing double API calls during the exit animation window.
- **Schema compliance**: BPP sanitises stored provider locations before publishing (`sanitizeLocations`) to strip non-Beckn-v2 address fields (`areaCode`, `door`, etc.) and convert GPS strings to GeoJSON Points.
- **WIB timezone**: All dates displayed in the Indonesian frontend use `Asia/Jakarta` (UTC+7) via the `fmtDate`/`fmtDateTime` helpers in `src/utils/date.js`.

---

## License

MIT
