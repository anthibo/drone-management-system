# Drone Delivery Management Backend

Single Go service exposing **REST**, **gRPC**, and **Thrift** APIs with JWT auth and Postgres persistence.

## Features
- JWT-based authentication (`/auth/token`) with roles: `admin`, `enduser`, `drone`
- REST + gRPC + Thrift transports on a shared service layer
- Postgres persistence with migrations
- Concurrency-safe job reservation
- ETA computation using Haversine distance + fixed drone speed

## Requirements
- Go 1.21+
- Postgres (local or Docker)

## Quick Start

### 1) Start Postgres + NATS
```bash
docker compose up -d
```

### 2) Run the server
```bash
export DATABASE_URL="postgres://drone:drone@localhost:5432/drone?sslmode=disable"
export JWT_SECRET="dev-secret"
export NATS_URL="nats://localhost:4222"

go run ./cmd/server
```

The service will listen on:
- REST: `:8080`
- gRPC: `:9090`
- Thrift: `:9091`

## Authentication

Issue a token:
```bash
curl -s -X POST http://localhost:8080/auth/token \
  -H "Content-Type: application/json" \
  -d '{"name":"alice","role":"enduser"}'
```

Use the token as a bearer:
```
Authorization: Bearer <token>
```

## REST Examples

Submit an order:
```bash
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"origin":{"lat":24.7136,"lng":46.6753},"destination":{"lat":24.7743,"lng":46.7386}}'
```

Reserve a job (drone role):
```bash
curl -s -X POST http://localhost:8080/drone/jobs/reserve \
  -H "Authorization: Bearer <token>"
```

## gRPC

The gRPC server uses a JSON codec instead of protobuf generation (no generated code required).
Clients must send `content-type: application/grpc+json` and use JSON payloads matching the structures in `internal/transport/mapper.go` and `internal/transport/grpcapi/types.go`.

## Thrift

The Thrift server uses the binary protocol and the IDL in `thrift/drone_delivery.thrift`.
Auth token is included in each request struct field `authToken`.

## Configuration
- `DATABASE_URL` (required)
- `JWT_SECRET` (required)
- `JWT_TTL` (default `1h`)
- `HTTP_ADDR` (default `:8080`)
- `GRPC_ADDR` (default `:9090`)
- `THRIFT_ADDR` (default `:9091`)
- `DRONE_SPEED_MPS` (default `15.0`)
- `MIGRATE_ON_START` (default `true`)
- `NATS_URL` (default `nats://127.0.0.1:4222`)
- `NATS_SUBJECT` (default `drone.events`)
- `OUTBOX_ENABLED` (default `true`)
- `OUTBOX_POLL_INTERVAL` (default `1s`)
- `OUTBOX_BATCH_SIZE` (default `50`)

## Tests
```bash
go test ./...
```

## Separate Outbox Worker
If you want event publishing decoupled from the API process:
1. Run API with `OUTBOX_ENABLED=false`
2. Run worker:
```bash
export DATABASE_URL="postgres://drone:drone@localhost:5432/drone?sslmode=disable"
export NATS_URL="nats://localhost:4222"
export OUTBOX_ENABLED=true
go run ./cmd/worker
```

## Notes
- Drones that mark themselves broken always create a handoff job even if fixed later.
- Endusers can withdraw orders before pickup; reserved orders are unassigned during withdrawal.
- The outbox worker provides **at-least-once** delivery. Consumers should be idempotent.

## Eventing (Outbox + NATS)
Order and drone state changes enqueue outbox events in Postgres.  
The outbox worker publishes to NATS on the configured subject, then marks events as published.
