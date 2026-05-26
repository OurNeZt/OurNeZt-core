FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o /out/ournezt-core ./cmd/core
RUN go install -tags "postgres file" github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.0

FROM alpine:3.20

RUN adduser -D -H -u 10001 appuser
WORKDIR /app
COPY --from=build /out/ournezt-core /app/ournezt-core
COPY --from=build /go/bin/migrate /usr/local/bin/migrate
COPY db/migrations /app/db/migrations
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh
USER appuser

EXPOSE 50051
ENTRYPOINT ["/app/docker-entrypoint.sh"]
