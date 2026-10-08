SHELL := /bin/sh

.PHONY: dev build test test-integration lint fmt backend frontend compose-up compose-down compose-logs migrate-up migrate-down

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

lint:
	go vet ./...
	npm --prefix web run lint

fmt:
	gofmt -w $$(find cmd internal tests -name '*.go' -type f)

backend:
	go run ./cmd/server

frontend:
	npm --prefix web run dev

compose-up:
	docker compose up --build

compose-down:
	docker compose down

compose-logs:
	docker compose logs -f

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down
