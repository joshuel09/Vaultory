# Vaultory backend. Targets map to the constitution's Quality Gates.
SHELL := /bin/bash
DB_URL ?= $(VAULTORY_DATABASE_URL)
MIGRATIONS := ./migrations

.PHONY: fmt vet build test test-unit test-integration gate migrate migrate-down run

fmt:
	gofmt -l -w .

vet:
	go vet ./...

build:
	go build ./...

test-unit:
	go test ./tests/unit/... ./internal/...

# Requires a real PostgreSQL instance; see quickstart.md.
test-integration:
	go test -tags=integration ./tests/integration/... ./tests/contract/...

test: test-unit test-integration

# The constitution's backend Quality Gates, in one command.
gate: vet build test

migrate:
	migrate -path $(MIGRATIONS) -database "$(DB_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS) -database "$(DB_URL)" down 1

run:
	go run ./cmd/vaultory-api
