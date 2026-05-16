# OurNeZt Core

`ournezt-core` is the backend service for OurNeZt. It owns business logic, PostgreSQL access, household finance calculations, CPF estimates, housing affordability, and internal gRPC contracts.

This folder is the first local build of the future `github.com/OurNeZt/ournezt-core` repository.

## Current Scope

- Go service entrypoint with gRPC health server
- Environment config loader
- Structured logging
- PostgreSQL connection helper
- Argon2id password hashing
- Secure session token generation
- Domain models for families, people, CPF, and housing
- Income, CPF, and housing affordability calculators
- Initial PostgreSQL migration
- Draft proto contracts for the web-to-core boundary
- `sqlc`, `buf`, Docker, and Make targets

## Local Run

Install Go 1.22+, then:

```sh
cp .env.example .env
go mod tidy
go test ./...
go run ./cmd/core
```

The service listens on `GRPC_ADDR`, defaulting to `:50051`.

## Database

Migrations live in `db/migrations`.

```sh
set DATABASE_URL=postgres://ournezt:ournezt@localhost:5432/ournezt?sslmode=disable
migrate -path db/migrations -database "%DATABASE_URL%" up
```

## Calculation Notes

CPF defaults are based on CPF Board 2026 public guidance:

- CPF contribution rates from 1 January 2026 for Singapore Citizens and SPRs from the third year onwards.
- Ordinary Wage ceiling of SGD 8,000 from 1 January 2026.
- CPF calculations are planning estimates and should stay labelled as estimates in the web app.

Official references:

- https://www.cpf.gov.sg/employer/employer-obligations/how-much-cpf-contributions-to-pay
- https://www.cpf.gov.sg/service/article/what-is-the-ordinary-wage-ow-ceiling

## Next Build Steps

1. Install Go, `buf`, `protoc-gen-go`, `protoc-gen-go-grpc`, `sqlc`, and `golang-migrate`.
2. Generate protobuf and SQL code.
3. Implement concrete gRPC handlers around the service layer.
4. Add repository implementations backed by generated `sqlc` queries.

