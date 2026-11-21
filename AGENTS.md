# Antigravity Engineering Guidelines — Fatman

This file is the authoritative workspace rule set for Antigravity and all contributors working in `clickTwice26/fatman`. Every agent, subagent, and contributor MUST follow these rules without exception.

---

## 1. Product Vision & Community Ethos

**Fatman** is a supportive, accountability-driven community for people on a weight-loss journey to:
- Share progress, daily wins, struggles, and encouragement in a judgment-free community feed.
- Communicate in real time via community rooms, quest squad chats, and 1:1 accountability buddy messaging.
- Join and complete collaborative **Weight-Loss Quests** (solo, duo, and squad challenges) with daily check-ins, streaks, milestones, and positive reinforcement.

**UX & Domain Guardrails:**
- Prioritize encouragement, consistency, and healthy habits over shaming or extreme dieting.
- Make sensitive metrics (starting weight, current weight, goal weight, check-in photos) **privacy-configurable** per user (`public`, `squads_only`, `private`).

---

## 2. Canonical Tech Stack & Latest-Version Policy

**Hard Rule — Always Use the Latest Stable Versions:**
- Before adding any Go module, npm package, or Docker base image, check the latest stable release (`npm view <pkg> version`, `go list -m -versions <mod>`, or Docker Hub tags) and install the newest stable version (avoid alpha/beta/rc unless explicitly approved).
- Never use deprecated packages (for example, do **not** install legacy `hugeicons-react`; always use `@hugeicons/react` + `@hugeicons/core-free-icons`).

| Layer | Technology | Rules |
| :--- | :--- | :--- |
| **Frontend** | Next.js (App Router) + React + TypeScript | Strict TS (`strict: true`), Server Components by default, Client Components only for interactive islands. |
| **UI Library** | Material UI (`@mui/material`, `@mui/material-nextjs`, `@emotion/*`) | Centralize theme tokens in `apps/web/src/theme/`. No ad-hoc inline magic colors. |
| **Icons** | Hugeicons (`@hugeicons/react`, `@hugeicons/core-free-icons`) | Use Hugeicons consistently across the UI. Do not mix icon packs. |
| **Typography** | **Onest** (`next/font/google`) | Load via `next/font/google` and bind `--font-onest` into the MUI theme `typography.fontFamily`. |
| **Backend** | Go (latest installed toolchain) | Standard library first (`net/http`, `log/slog`, `context`) + minimal vetted libraries (`chi`, `pgx/v5`, `go-redis/v9`, `coder/websocket`, `golang-jwt/jwt/v5`, `validator/v10`, `goose/v3`). |
| **Database** | PostgreSQL (Dockerized, latest stable) | Versioned SQL migrations only. Strictly parameterized queries (zero SQL string interpolation). Note: PG 18+ mounts data at `/var/lib/postgresql`. |
| **NoSQL / Cache** | Redis (Dockerized, latest stable) | Required for 4 concerns: **Caching**, **Authentication**, **Validation**, and **Rate Limiting** (see Section 4). |

---

## 3. Professional Monorepo & Architecture Structure

Follow this repository layout strictly. Do not scatter files at the root.

```text
fatman/
├── AGENTS.md                        # Workspace agent guidelines (this file)
├── Makefile                         # Unified dev, test, lint, migrate, verify targets
├── README.md
├── infra/
│   └── docker/
│       └── docker-compose.yml       # PostgreSQL + Redis (+ healthchecks & named volumes)
├── services/
│   └── api/                         # Go backend service
│       ├── cmd/
│       │   └── api/main.go          # Minimal entrypoint: wiring & graceful shutdown only
│       ├── internal/
│       │   ├── config/              # Typed env configuration & validation
│       │   ├── domain/              # Pure domain entities, value objects, domain errors
│       │   ├── repository/          # Interfaces + PostgreSQL (pgx) & Redis implementations
│       │   ├── service/             # Business logic & use-case orchestration (depends on interfaces)
│       │   ├── transport/
│       │   │   ├── http/            # Router, handlers, DTOs, middleware (auth, rate-limit, CORS, logging)
│       │   │   └── ws/              # WebSocket hub, room subscriptions, Redis Pub/Sub bridge
│       │   └── infrastructure/
│       │       ├── postgres/        # Connection pool & health check
│       │       └── redis/           # Redis client, scripts, key builders & health check
│       └── migrations/              # Ordered, reversible SQL migrations (goose)
├── apps/
│   └── web/                         # Next.js App Router frontend
│       ├── src/
│       │   ├── app/                 # Routes, layouts, loading/error boundaries
│       │   ├── theme/               # MUI ThemeProvider, Onest font, color palette, component overrides
│       │   ├── components/          # Reusable domain-agnostic UI primitives
│       │   ├── features/            # Feature slices (auth, profile, feed, quests, chat, notifications)
│       │   └── lib/                 # Typed API client, WS client, validation schemas, formatters
│       └── e2e/                     # Playwright end-to-end test suites
└── docs/                            # ADRs, API contracts, verification notes
```

### Dependency Direction (Backend Clean Architecture)
- `transport` → `service` → `domain` ← `repository`
- `domain` has **zero** external infrastructure imports (`pgx`, `go-redis`, `net/http` are forbidden in `domain`).
- `transport` handles HTTP/WS parsing, DTO validation, and status codes; business rules live exclusively in `service` and `domain`.
- `repository` implements interfaces declared by `service`/`domain` and handles SQL/Redis persistence only.

---

## 4. Redis Responsibility Contract

Redis is a first-class architectural component with four explicit responsibilities:

1. **Caching:** Profile summaries, paginated feed slices, active quest metadata, and quest leaderboards (`ZSET`). Every cache entry MUST have a bounded TTL and explicit invalidation on write.
2. **Authentication:** Refresh-token session records (supporting token rotation, reuse detection, multi-device logout), plus access-token revocation denylist (`jti`) until natural expiry.
3. **Validation:** Short-lived email verification OTPs, password-reset tokens, registration/action idempotency keys, and temporary reservation locks.
4. **Rate Limiting:** Distributed sliding-window / atomic Lua rate limiters for auth endpoints, post/comment creation, chat bursts, and quest check-ins. Always return HTTP `429 Too Many Requests` with `Retry-After` and `X-RateLimit-*` headers.

**Key Naming Convention:**
`fatman:{env}:{scope}:{entity}:{identifier}`
- Examples:
  - `fatman:dev:auth:session:{sessionID}`
  - `fatman:dev:auth:deny:{jti}`
  - `fatman:dev:val:otp:{purpose}:{userID}`
  - `fatman:dev:rl:{route}:{subject}`
  - `fatman:dev:cache:quest:{questID}:leaderboard`
- Never store keys without a TTL (except explicitly managed structures with cleanup policies).

---

## 5. Git & Commit Discipline (Chunk-by-Chunk, Feature-by-Feature)

**NEVER commit large multi-feature dumps.** Every change MUST be broken down into small, cohesive, independently verifiable chunks.

1. **One Concern Per Commit:**
   - Infrastructure setup, a single migration + repository, a single service + unit tests, an HTTP endpoint + integration test, or a single frontend feature view each belong in their own focused commit.
   - Keep commits reviewable and logically atomic (a broken or half-finished state must never be committed).
2. **Conventional Commits Format:**
   - `<type>(<scope>): <concise imperative summary>`
   - Allowed types: `feat`, `fix`, `test`, `refactor`, `chore`, `docs`, `perf`, `ci`, `build`.
   - Examples:
     - `chore(infra): add Docker Compose for PostgreSQL and Redis`
     - `feat(api): add Redis sliding-window rate limiter middleware`
     - `test(api): add integration tests for quest check-in streaks`
     - `feat(web): implement MUI theme with Onest font and Hugeicons wrapper`
3. **Commit Only Verified Code:**
   - Never commit untested code, failing tests, TypeScript errors, lint errors, or `.env` secrets.

---

## 6. Test-Driven & Verification-Driven Development (TDD / VDD)

Every feature follows the **Red → Green → Refactor → Verify** loop:

1. **Red:** Write failing unit and/or integration tests specifying the exact contract and edge cases *before* writing implementation code.
2. **Green:** Implement the minimal clean code required to make the tests pass.
3. **Refactor:** Improve structure, readability, and error handling while keeping all tests green.
4. **Verify:** Run the full verification suite for the touched area before committing.

### Mandatory Verification Gates Before Every Commit

**Backend (`services/api`):**
- `go test -race -count=1 ./...` — all unit and integration tests pass with the race detector enabled.
- `go vet ./...` — zero warnings.
- `go build ./...` — compiles cleanly.
- Database migrations verified up and down against live Docker PostgreSQL.

**Frontend (`apps/web`):**
- `npm run typecheck` (`tsc --noEmit`) — zero TypeScript errors.
- `npm run lint` — zero lint errors.
- `npm run test` (Vitest + Testing Library) — all unit/component tests pass.
- `npm run build` — production Next.js build succeeds cleanly.
- `npm run test:e2e` (Playwright) — required at the completion of every user-facing feature milestone.

**Verification Honesty Rule:**
- Never claim a feature works without running real verification commands and checking actual outputs (logs, HTTP responses, test summaries, or browser rendering).

---

## 7. Security & Code Quality Standards

- **Passwords:** Hash with `argon2id` (`golang.org/x/crypto/argon2`), unique per-user cryptographic salt, constant-time verification.
- **Tokens & Cookies:** Short-lived JWT access tokens (e.g., 15 min) + opaque or JWT refresh tokens stored in `HttpOnly`, `SameSite=Lax`, `Secure` (in prod) cookies and backed by Redis sessions with rotation.
- **Input Validation:** Validate all request payloads at the boundary (Go `validator/v10` + frontend Zod schemas). Reject unknown/oversized payloads (`http.MaxBytesReader`).
- **Error Handling:** Consistent JSON error envelope `{ "error": { "code": "...", "message": "...", "details": ... } }`. Never leak stack traces, SQL errors, or internal secrets to clients.
- **Observability:** Structured JSON logging via Go `log/slog` with request IDs (`X-Request-ID`) and latency metrics.
- **Cleanliness:** No dead code, no commented-out blocks, no `any` in TypeScript, no ignored errors (`_ = err`) in Go.
