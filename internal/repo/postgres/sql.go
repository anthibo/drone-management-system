package postgres

const orderSelectByIDSQL = `
SELECT id, user_id, origin_lat, origin_lng, dest_lat, dest_lng, status,
       assigned_drone_id, handoff_origin_lat, handoff_origin_lng,
       created_at, updated_at, reserved_at, picked_up_at, delivered_at, failed_at, failure_reason
FROM orders
WHERE id = $1
`

const orderSelectByIDForUpdateSQL = orderSelectByIDSQL + " FOR UPDATE"

const orderListSQL = `
SELECT id, user_id, origin_lat, origin_lng, dest_lat, dest_lng, status,
       assigned_drone_id, handoff_origin_lat, handoff_origin_lng,
       created_at, updated_at, reserved_at, picked_up_at, delivered_at, failed_at, failure_reason
FROM orders
WHERE ($1::text IS NULL OR status = $1)
ORDER BY created_at
LIMIT $2 OFFSET $3
`

const orderInsertSQL = `
INSERT INTO orders (
  id, user_id, origin_lat, origin_lng, dest_lat, dest_lng, status,
  assigned_drone_id, handoff_origin_lat, handoff_origin_lng,
  created_at, updated_at, reserved_at, picked_up_at, delivered_at, failed_at, failure_reason
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,
  $8,$9,$10,
  $11,$12,$13,$14,$15,$16,$17
)
`

const orderUpdateSQL = `
UPDATE orders SET
  user_id = $1,
  origin_lat = $2,
  origin_lng = $3,
  dest_lat = $4,
  dest_lng = $5,
  status = $6,
  assigned_drone_id = $7,
  handoff_origin_lat = $8,
  handoff_origin_lng = $9,
  updated_at = $10,
  reserved_at = $11,
  picked_up_at = $12,
  delivered_at = $13,
  failed_at = $14,
  failure_reason = $15
WHERE id = $16
`

const orderReserveSQL = `
SELECT id, user_id, origin_lat, origin_lng, dest_lat, dest_lng, status,
       assigned_drone_id, handoff_origin_lat, handoff_origin_lng,
       created_at, updated_at, reserved_at, picked_up_at, delivered_at, failed_at, failure_reason
FROM orders
WHERE status = ANY($1)
  AND assigned_drone_id IS NULL
ORDER BY created_at
LIMIT 1
FOR UPDATE SKIP LOCKED
`

const droneSelectByIDSQL = `
SELECT id, status, last_lat, last_lng, last_heartbeat_at, current_order_id, created_at, updated_at
FROM drones
WHERE id = $1
`

const droneSelectByIDForUpdateSQL = droneSelectByIDSQL + " FOR UPDATE"

const droneInsertSQL = `
INSERT INTO drones (
  id, status, last_lat, last_lng, last_heartbeat_at, current_order_id, created_at, updated_at
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8
)
`

const droneUpdateSQL = `
UPDATE drones SET
  status = $1,
  last_lat = $2,
  last_lng = $3,
  last_heartbeat_at = $4,
  current_order_id = $5,
  updated_at = $6
WHERE id = $7
`

const droneListSQL = `
SELECT id, status, last_lat, last_lng, last_heartbeat_at, current_order_id, created_at, updated_at
FROM drones
ORDER BY id
`

const outboxInsertSQL = `
INSERT INTO outbox_events (
  id, event_type, aggregate_type, aggregate_id, payload, occurred_at
) VALUES ($1,$2,$3,$4,$5,$6)
`

const outboxFetchPendingSQL = `
SELECT id, event_type, aggregate_type, aggregate_id, payload, occurred_at
FROM outbox_events
WHERE published_at IS NULL
ORDER BY occurred_at
LIMIT $1
`

const outboxMarkPublishedSQL = `
UPDATE outbox_events
SET published_at = now()
WHERE id = ANY($1::uuid[])
`
