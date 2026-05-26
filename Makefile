.PHONY: build test run migrate-up migrate-down sqlc proto tidy

build:
	go build -o bin/ournezt-core ./cmd/core

test:
	go test ./...

run:
	go run ./cmd/core

migrate-up:
	migrate -path db/migrations -database "$$DATABASE_URL" up

migrate-down:
	migrate -path db/migrations -database "$$DATABASE_URL" down 1

sqlc:
	sqlc generate

proto:
	buf generate

tidy:
	go mod tidy

