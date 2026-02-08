package events

import "context"

type NoopPublisher struct{}

func (NoopPublisher) Publish(ctx context.Context, event Event) error {
	return nil
}

func (NoopPublisher) Close() error {
	return nil
}

