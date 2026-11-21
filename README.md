# Fatman — Weight-Loss Community & Collaborative Quests

Fatman is a supportive, accountability-first community platform for people on a weight-loss journey to connect, encourage each other, track milestones, and complete weight-loss quests together.

## Tech Stack

- **Frontend:** Next.js (App Router, TypeScript), Material UI (MUI), Hugeicons, **Onest** typography
- **Backend:** Go (Clean Architecture, REST + WebSockets)
- **Primary Database:** PostgreSQL (Dockerized)
- **NoSQL / In-Memory Store:** Redis (Dockerized) — caching, authentication sessions, validation tokens/idempotency, and sliding-window rate limiting

## Repository Structure

```text
fatman/
├── AGENTS.md                 # Antigravity engineering guidelines & workflow rules
├── apps/
│   └── web/                  # Next.js + MUI + Hugeicons frontend
├── services/
│   └── api/                  # Go backend API + WebSocket hub + SQL migrations
├── infra/
│   └── docker/               # Docker Compose (PostgreSQL + Redis) & container configs
├── docs/                     # Architecture decisions & API documentation
└── Makefile                  # Unified developer & verification commands
```

## Engineering Principles

1. **Chunk-by-chunk, feature-by-feature commits** — small, atomic, verifiable Conventional Commits.
2. **Test-Driven & Verification-Driven** — write failing tests first, verify green builds and passing checks before every commit.
3. **Latest Stable Dependencies** — always use the latest stable versions of Go, Node.js, Next.js, MUI, Hugeicons, PostgreSQL, and Redis packages.

See [AGENTS.md](./AGENTS.md) for the complete engineering and Antigravity workflow guidelines.
