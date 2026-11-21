SHELL := /usr/bin/env bash
COMPOSE := docker compose -f infra/docker/docker-compose.yml

.PHONY: help infra-up infra-down infra-ps infra-logs api-dev api-test api-vet api-build web-dev web-test web-verify verify

help:
	@echo "Fatman developer commands:"
	@echo "  make infra-up      Start PostgreSQL 19, Redis 8, and Mailpit in Docker"
	@echo "  make infra-down    Stop Docker infrastructure"
	@echo "  make infra-ps      Show infrastructure container status"
	@echo "  make api-dev       Run Go API server locally"
	@echo "  make api-test      Run Go unit and integration tests with -race"
	@echo "  make api-vet       Run go vet across services/api"
	@echo "  make api-build     Build Go API binary"
	@echo "  make web-dev       Run Next.js frontend dev server"
	@echo "  make web-verify    Run Next.js typecheck, lint, unit tests, and build"
	@echo "  make verify        Run full backend and frontend verification suite"

infra-up:
	$(COMPOSE) up -d --wait

infra-down:
	$(COMPOSE) down

infra-ps:
	$(COMPOSE) ps

infra-logs:
	$(COMPOSE) logs -f

api-dev:
	cd services/api && go run ./cmd/api

api-test:
	cd services/api && go test -race -count=1 ./...

api-vet:
	cd services/api && go vet ./...

api-build:
	cd services/api && go build ./...

web-dev:
	cd apps/web && npm run dev

web-test:
	cd apps/web && npm run test

web-verify:
	cd apps/web && npm run typecheck && npm run lint && npm run test && npm run build

verify: api-vet api-test api-build web-verify
