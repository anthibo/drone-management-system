CREATE TABLE IF NOT EXISTS outbox_events (
  id uuid PRIMARY KEY,
  event_type text NOT NULL,
  aggregate_type text NOT NULL,
  aggregate_id text NOT NULL,
  payload jsonb NOT NULL,
  occurred_at timestamptz NOT NULL,
  published_at timestamptz NULL
);

CREATE INDEX IF NOT EXISTS idx_outbox_published ON outbox_events (published_at, occurred_at);
