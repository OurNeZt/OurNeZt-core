FROM golang:1.22-alpine AS build

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o /out/ournezt-core ./cmd/core

FROM alpine:3.20

RUN adduser -D -H -u 10001 appuser
WORKDIR /app
COPY --from=build /out/ournezt-core /app/ournezt-core
USER appuser

EXPOSE 50051
ENTRYPOINT ["/app/ournezt-core"]
