.PHONY: build test run migrate-up migrate-down sqlc proto tidy

DATABASE_URL ?=
MIGRATIONS_PATH ?= db/migrations

build:
	go build -o bin/ournezt-core ./cmd/core

test:
	go test ./...

run:
	go run ./cmd/core

guard-database-url:
ifeq ($(strip $(DATABASE_URL)),)
	$(error DATABASE_URL is required. Example: make migrate-up DATABASE_URL="postgres://ournezt:ournezt@localhost:5432/ournezt?sslmode=disable")
endif

migrate-up: guard-database-url
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up

migrate-down: guard-database-url
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down 1

sqlc:
	sqlc generate

proto:
	buf generate

tidy:
	go mod tidy

