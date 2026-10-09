SHELL := /bin/sh

APEXVOID_APPS_NETWORK ?= apexvoid-apps
APEXVOID_DATA_NETWORK ?= apexvoid-data

.PHONY: dev build test test-integration test-external-docker lint fmt backend frontend ensure-compose-networks compose-up compose-down compose-logs migrate-up migrate-down

dev: compose-up

build:
	go build ./...
	npm --prefix web run build

test:
	go test ./...
	npm --prefix web run typecheck
	npm --prefix web run test

test-integration:
	go test -tags=integration ./tests/integration

test-external-docker:
	./scripts/test-external-module-docker.sh

lint:
	go vet ./...
	npm --prefix web run lint

fmt:
	gofmt -w $$(find cmd internal tests -name '*.go' -type f)

backend:
	go run ./cmd/server

frontend:
	npm --prefix web run dev

ensure-compose-networks:
	@for network in "$(APEXVOID_APPS_NETWORK)" "$(APEXVOID_DATA_NETWORK)"; do \
		if docker network inspect "$$network" >/dev/null 2>&1; then \
			echo "Docker network already exists: $$network"; \
		else \
			echo "Creating Docker network: $$network"; \
			docker network create "$$network" >/dev/null; \
		fi; \
	done

compose-up: ensure-compose-networks
	docker compose up --build -d --force-recreate

compose-down:
	docker compose down -v --remove-orphans

compose-logs:
	docker compose logs -f

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down
