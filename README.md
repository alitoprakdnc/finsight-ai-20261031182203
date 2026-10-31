# FinSight AI

FinSight AI is a secure, multi‑tenant SaaS platform that automates the ingestion, OCR extraction, AI‑enhanced categorisation and bookkeeping of financial documents (invoices, receipts, contracts). It provides a modern React dashboard for finance teams, a robust Go‑based API with OpenAPI spec, server‑less task processing, webhook notifications, role‑based access control, audit logging and full integration with cloud secret managers and storage. The system eliminates manual data entry, reduces errors, and keeps sensitive financial data encrypted at rest and in transit.

## Technologies

- Go (Gin)
- PostgreSQL
- Redis
- Docker & Docker‑Compose
- Kubernetes (Helm)
- React + TypeScript (Next.js)
- Tailwind CSS
- Tesseract OCR
- Ollama (local LLM) / OpenAI API
- Google Cloud Secret Manager
- Google Cloud Tasks (fallback to BullMQ)
- AWS S3‑compatible storage (MinIO)
- OpenAPI 3.0
- JWT + OIDC
- Cypress (E2E)
- Go test + testify

## Features

- Secure multi‑tenant document upload with client‑side encryption
- Automatic OCR extraction using Tesseract, with language auto‑detect
- LLM‑driven data enrichment: vendor name resolution, line‑item categorisation, tax calculation
- Configurable approval workflows and role‑based permissions (Admin, Accountant, Viewer)
- Audit‑ready immutable log stored in PostgreSQL with tamper‑evidence
- Webhook engine with constant‑time HMAC validation for downstream systems
- RESTful API with full OpenAPI 3.0 spec and interactive Swagger UI
- Health‑check endpoint (/api/health) exposing liveness/readiness metrics
- Serverless task queue (Google Cloud Tasks or BullMQ) for scalable processing
- Integration with Google Cloud Secret Manager for API keys and DB credentials
- Export to CSV/Excel and direct push to popular accounting SaaS via webhooks
- Responsive dashboard with charts (spending trends, pending approvals) and searchable tables
- Dark / Light theme, accessibility‑focused UI
- CI/CD ready GitHub Actions workflow that runs lint, unit, integration and security scans

## Architecture

The system is split into three major layers:

1. **Frontend (Next.js)** – Serves a SPA with authentication (OAuth2/OIDC), role‑based UI rendering, file upload component (uses pre‑signed URLs from the backend), dashboards, and settings pages. Tailwind CSS provides a consistent design system.
2. **Backend API (Go + Gin)** – Exposes a versioned REST API (`/api/v1`). Core modules:
   - `auth` – JWT issuance, OIDC integration, middleware for RBAC.
   - `documents` – Handles upload metadata, generates pre‑signed S3 URLs, stores reference records.
   - `processing` – Pushes a task to Google Cloud Tasks (or BullMQ fallback) containing the document ID.
   - `worker` – Consumes tasks, runs Tesseract OCR, calls the LLM service (local Ollama or OpenAI), writes extracted data back to DB, triggers webhooks.
   - `audit` – Writes immutable events to `audit_logs` table, also streams to Redis for real‑time UI updates.
   - `webhook` – Validates inbound HMAC signatures, stores subscription info, retries on failure.
   - `health` – Liveness/readiness probes, metrics endpoint (`/metrics`).
3. **Infrastructure** – Docker‑Compose for local dev, Helm charts for K8s production. Services include PostgreSQL, Redis, MinIO (S3), Google Cloud Tasks emulator, and optional Ollama container. Secrets are fetched at runtime from Google Cloud Secret Manager via the Go client.

All communication uses HTTPS. Files are encrypted at rest in MinIO with server‑side encryption. JWTs are signed with RSA keys stored in Secret Manager. Rate‑limiting and request validation are enforced globally.

Data flow: User → Frontend → API (`/documents`) → Pre‑signed URL → Upload to MinIO → API creates `document` record → Enqueue task → Worker OCR → Worker LLM → DB update → Webhook → UI real‑time update via Redis Pub/Sub.

## Product Design

FinSight AI adopts a minimalist, enterprise‑grade design system built on Tailwind CSS:
- **Typography**: Inter (body) and Roboto Slab (headings) with responsive scaling.
- **Color palette**: Primary #0A74DA, secondary #14A44D, neutrals #F5F7FA (light) / #1F2937 (dark), accent #F59E0B.
- **Spacing**: 4‑px base unit, consistent margin/padding utilities.
- **Components**: Card, Table with sortable columns, Modal, Drawer, Toast, Avatar, Badge, Progress Bar, Chart (Recharts), FileUploader (drag‑drop + preview), Permission Switcher.
- **Interaction states**: Focus rings, hover/active shadows, disabled opacity.
- **Responsive**: Mobile‑first breakpoints (sm 640px, md 768px, lg 1024px). Dashboard collapses side‑nav into hamburger on <md.
- **Accessibility**: ARIA labels, keyboard navigation, color contrast ≥4.5:1.
- **Dark mode** toggled via user preference, persisted in JWT claims.
The UI is built as a component library under `src/components` and reused across pages (`/dashboard`, `/documents`, `/settings`). Loading skeletons, empty states with friendly illustrations, and error toasts provide polished feedback.

## Testing

Testing is layered:
- **Unit**: Go packages tested with `go test` + `testify` for business logic, OCR wrapper, LLM client, webhook validator.
- **Integration**: Docker‑Compose spins up API, PostgreSQL, Redis, MinIO; tests run against real services using `httptest` and `sqlx`.
- **Contract**: OpenAPI spec validated with `speccy` and generated Go client stubs; CI ensures spec matches implementation.
- **E2E**: Cypress scripts cover user flows – login, document upload, processing status, approval, webhook subscription.
- **Security**: GitHub Actions run `gosec`, `npm audit`, and OWASP Dependency‑Check.
- **Performance**: Locust scenarios for upload throughput and task queue latency.
All tests run in CI on every PR, with coverage thresholds (80%).

## Status

Generated by QuarkForge. Review and test before deployment. Automated test status is recorded in the generation logs.
