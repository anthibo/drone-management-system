package postgres

import (
	"context"
	"time"

	"penny-assesment/internal/events"
)

func (s *Store) FetchPending(ctx context.Context, limit int) ([]events.Event, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, outboxFetchPendingSQL, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var evts []events.Event
	for rows.Next() {
		evt, err := scanOutboxEvent(rows)
		if err != nil {
			return nil, err
		}
		evts = append(evts, evt)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return evts, nil
}

func (s *Store) MarkPublished(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := s.pool.Exec(ctx, outboxMarkPublishedSQL, ids)
	return err
}

func scanOutboxEvent(row pgxRow) (events.Event, error) {
	var payload []byte
	var occurredAt time.Time
	var evt events.Event
	if err := row.Scan(&evt.ID, &evt.Type, &evt.AggregateType, &evt.AggregateID, &payload, &occurredAt); err != nil {
		return events.Event{}, err
	}
	evt.Payload = payload
	evt.OccurredAt = occurredAt
	return evt, nil
}

type pgxRow interface {
	Scan(dest ...any) error
}
