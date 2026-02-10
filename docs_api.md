# Drone Delivery Management Backend — API Documentation

This service exposes **REST**, **gRPC**, and **Thrift** APIs backed by a shared Go service layer and Postgres.

- REST base: `http://localhost:8080` (in my run: `http://127.0.0.1:18080`)
- gRPC: `:9090` (in my run: `:19090`)
- Thrift: `:9091` (in my run: `:19091`)

## Authentication

### Issue JWT
`POST /auth/token`

Request:
```json
{ "name": "alice", "role": "enduser" }
```
- `role` must be one of: `admin | enduser | drone`

Response:
```json
{ "token": "<jwt>", "expires_at": "<rfc3339>" }
```

Use it on all other REST endpoints:
```
Authorization: Bearer <jwt>
```

---

## REST Endpoints

### Enduser

#### Submit order
`POST /orders`

Body:
```json
{
  "origin": {"lat": 24.7136, "lng": 46.6753},
  "destination": {"lat": 24.7743, "lng": 46.7386}
}
```

Response (201): `OrderResponse`

#### Withdraw order (only before pickup)
`POST /orders/{id}/withdraw`

Response (200): `OrderResponse`

Errors:
- 409 `precondition_failed` if already picked up / not withdrawable.

#### Get order details (progress + location + ETA)
`GET /orders/{id}`

Response (200):
```json
{
  "order": { /* OrderResponse */ },
  "current_location": {"lat": 24.72, "lng": 46.68},
  "eta_seconds": 563
}
```

---

### Drone

#### Reserve a job
`POST /drone/jobs/reserve`

Response (200): `OrderResponse`

Errors:
- 404 `no_job` if no available jobs.

#### Pick up an order
`POST /drone/orders/{id}/pickup`

Response (200): `OrderResponse`

#### Mark delivered
`POST /drone/orders/{id}/deliver`

Response (200): `OrderResponse`

#### Mark failed
`POST /drone/orders/{id}/fail`

Body:
```json
{ "reason": "battery" }
```

Response (200): `OrderResponse`

#### Mark drone broken
`POST /drone/broken`

Response (200): `DroneResponse`

Business rule: when a drone is marked broken while carrying an order, the order becomes `HANDOFF_REQUESTED`, the drone assignment is cleared, and `handoff_origin` is set to the drone last known location.

#### Heartbeat + location update + status
`POST /drone/heartbeat`

Body:
```json
{ "lat": 24.72, "lng": 46.68 }
```

Response (200):
```json
{ "drone": { /* DroneResponse */ }, "current_order": { /* OrderViewResponse */ } }
```

#### Get current assigned order
`GET /drone/orders/current`

Response (200): `OrderViewResponse`

---

### Admin

#### List orders (bulk)
`GET /admin/orders?status=&limit=&offset=`

Response (200): `OrderViewResponse[]`

#### Update order origin/destination
`PATCH /admin/orders/{id}`

Body:
```json
{ "origin": {"lat": 1, "lng": 1}, "destination": {"lat": 2, "lng": 2} }
```

Response (200): `OrderResponse`

#### List drones
`GET /admin/drones`

Response (200): `DroneResponse[]`

#### Mark drone broken/fixed
`POST /admin/drones/{id}/broken`
`POST /admin/drones/{id}/fixed`

Response (200): `DroneResponse`

---

## Data Types (REST)

### Location
```json
{ "lat": 0.0, "lng": 0.0 }
```

### OrderResponse
```json
{
  "id": "uuid",
  "user_id": "string",
  "origin": {"lat": 0, "lng": 0},
  "destination": {"lat": 0, "lng": 0},
  "status": "CREATED|RESERVED|PICKED_UP|DELIVERED|FAILED|WITHDRAWN|HANDOFF_REQUESTED",
  "assigned_drone_id": "string?",
  "handoff_origin": {"lat": 0, "lng": 0}?,
  "created_at": "rfc3339",
  "updated_at": "rfc3339",
  "reserved_at": "rfc3339?",
  "picked_up_at": "rfc3339?",
  "delivered_at": "rfc3339?",
  "failed_at": "rfc3339?",
  "failure_reason": "string?"
}
```

### DroneResponse
```json
{
  "id": "string",
  "status": "ACTIVE|BROKEN",
  "last_location": {"lat": 0, "lng": 0}?,
  "last_heartbeat_at": "rfc3339?",
  "current_order_id": "uuid?",
  "created_at": "rfc3339",
  "updated_at": "rfc3339"
}
```

---

## gRPC

- Proto: `proto/drone_delivery.proto`
- Services: `AuthService`, `OrderService`, `DroneService`, `AdminService`
- Auth is passed via **Bearer token metadata** (see `internal/transport/grpcapi`).

---

## Thrift

- IDL: `thrift/drone_delivery.thrift`
- Auth token is included in request structs field `authToken`.

