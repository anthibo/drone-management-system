package nats

import (
	"context"
	"encoding/json"

	"github.com/nats-io/nats.go"

	"penny-assesment/internal/events"
)

type Publisher struct {
	nc     *nats.Conn
	subject string
}

func New(url, subject string) (*Publisher, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	if subject == "" {
		subject = "drone.events"
	}
	return &Publisher{nc: nc, subject: subject}, nil
}

func (p *Publisher) Publish(ctx context.Context, event events.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.nc.Publish(p.subject, data)
}

func (p *Publisher) Close() error {
	if p.nc != nil {
		p.nc.Close()
	}
	return nil
}

var _ events.Publisher = (*Publisher)(nil)

