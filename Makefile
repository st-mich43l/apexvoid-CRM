SHELL := /bin/sh

.PHONY: dev build test lint fmt backend frontend compose-up compose-down compose-logs

dev: compose-up

build:
	go build ./...
	npm --prefix web run build

test:
	go test ./...
	npm --prefix web run typecheck

lint:
	go vet ./...
	npm --prefix web run lint

fmt:
	gofmt -w $$(find cmd internal -name '*.go' -type f)

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
