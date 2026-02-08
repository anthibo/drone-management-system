package events

import (
	"context"
	"log"
	"time"
)

type Publisher interface {
	Publish(ctx context.Context, event Event) error
	Close() error
}

type OutboxRepository interface {
	FetchPending(ctx context.Context, limit int) ([]Event, error)
	MarkPublished(ctx context.Context, ids []string) error
}

type OutboxWorker struct {
	Repo         OutboxRepository
	Publisher    Publisher
	PollInterval time.Duration
	BatchSize    int
	Logger       *log.Logger
}

func (w *OutboxWorker) Start(ctx context.Context) error {
	if w.Logger == nil {
		w.Logger = log.Default()
	}
	if w.PollInterval <= 0 {
		w.PollInterval = time.Second
	}
	if w.BatchSize <= 0 {
		w.BatchSize = 50
	}

	ticker := time.NewTicker(w.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			evts, err := w.Repo.FetchPending(ctx, w.BatchSize)
			if err != nil {
				w.Logger.Printf("outbox fetch error: %v", err)
				continue
			}
			if len(evts) == 0 {
				continue
			}
			published := make([]string, 0, len(evts))
			for _, evt := range evts {
				if err := w.Publisher.Publish(ctx, evt); err != nil {
					w.Logger.Printf("publish error id=%s type=%s: %v", evt.ID, evt.Type, err)
					continue
				}
				published = append(published, evt.ID)
			}
			if err := w.Repo.MarkPublished(ctx, published); err != nil {
				w.Logger.Printf("mark published error: %v", err)
			}
		}
	}
}

