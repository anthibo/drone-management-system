# Drone Delivery Management Backend

A single Go service exposing **REST**, **gRPC**, and **Thrift** APIs with **JWT auth**, **Postgres** persistence, and **NATS eventing** via the **transactional outbox pattern**.

![System Architecture](./assets/image.png)

## What this service does

### Roles
- **enduser**: create orders, withdraw before pickup, track progress + location + ETA
- **drone**: reserve jobs, pickup, deliver/fail, heartbeat (location + status), mark broken
- **admin**: list orders (bulk), update origin/destination, list drones, mark drones broken/fixed

### Core ideas
- **One service layer**: REST/gRPC/Thrift are thin transports over the same business logic.
- **Concurrency-safe reservation**: reservation uses DB locking (`FOR UPDATE SKIP LOCKED`).
- **ETA**: Haversine distance + fixed drone speed (`DRONE_SPEED_MPS`).
- **Events**: order/drone changes are written to Postgres outbox rows and published to NATS (at-least-once).

---

## Requirements
- Go **1.21+**
- Docker (for Postgres + NATS)

---

## Quick Start (local)

### 1) Start Postgres + NATS
```bash
docker compose up -d
```

> Note: `docker-compose.yml` maps Postgres to host port **65432** (to avoid clashing with local Postgres).

### 2) Run the server
```bash
export DATABASE_URL="postgres://drone:drone@127.0.0.1:65432/drone?sslmode=disable"
export JWT_SECRET="dev-secret"
export NATS_URL="nats://127.0.0.1:4222"

# Optional: avoid collisions
# export HTTP_ADDR=":18080" GRPC_ADDR=":19090" THRIFT_ADDR=":19091"

go run ./cmd/server
```

Default ports:
- REST: `:8080`
- gRPC: `:9090`
- Thrift: `:9091`

---

## Postman

Import both:
- `tools/postman/drone.postman_collection.json`
- `tools/postman/drone.postman_environment.json`

Then run:
1) Auth → Issue Token (enduser)
2) Auth → Issue Token (drone)
3) Auth → Issue Token (admin)
4) Enduser → Submit Order (stores `ORDER_ID` automatically)

---

## REST (curl) smoke test

### Issue a token
```bash
curl -s -X POST http://127.0.0.1:8080/auth/token \
  -H "Content-Type: application/json" \
  -d '{"name":"alice","role":"enduser"}'
```

---

## Watch NATS events

The `nats:2.10` server image does **not** include the `nats` CLI. Use a temporary CLI container:

```bash
docker run --rm -it --network drone-management-system_default natsio/nats-box:latest \
  nats sub drone.events --server nats://drone-management-system-nats-1:4222
```

Trigger any state change (submit/reserve/pickup/deliver/broken) and you’ll see events.

---

## gRPC

This gRPC server uses a **custom JSON codec** (see `internal/transport/grpcapi/codec.go`) instead of protobuf messages.

Implications:
- **grpcurl/Postman gRPC won’t work** (no reflection + messages are not `proto.Message`).
- Use a small Go client that forces the `json` codec.

---

## Thrift

- IDL: `thrift/drone_delivery.thrift`
- Protocol: binary (server uses framed transport)
- Auth token is included in request struct field `authToken`

---

## Tests
```bash
go test ./...
```
