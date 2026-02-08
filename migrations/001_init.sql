CREATE TABLE IF NOT EXISTS orders (
  id uuid PRIMARY KEY,
  user_id text NOT NULL,
  origin_lat double precision NOT NULL,
  origin_lng double precision NOT NULL,
  dest_lat double precision NOT NULL,
  dest_lng double precision NOT NULL,
  status text NOT NULL,
  assigned_drone_id text NULL,
  handoff_origin_lat double precision NULL,
  handoff_origin_lng double precision NULL,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  reserved_at timestamptz NULL,
  picked_up_at timestamptz NULL,
  delivered_at timestamptz NULL,
  failed_at timestamptz NULL,
  failure_reason text NULL
);

CREATE TABLE IF NOT EXISTS drones (
  id text PRIMARY KEY,
  status text NOT NULL,
  last_lat double precision NULL,
  last_lng double precision NULL,
  last_heartbeat_at timestamptz NULL,
  current_order_id uuid NULL,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_orders_status_created ON orders (status, created_at);
CREATE INDEX IF NOT EXISTS idx_orders_user ON orders (user_id);
CREATE INDEX IF NOT EXISTS idx_orders_assigned_drone ON orders (assigned_drone_id);
CREATE INDEX IF NOT EXISTS idx_drones_status ON drones (status);
CREATE INDEX IF NOT EXISTS idx_drones_current_order ON drones (current_order_id);

