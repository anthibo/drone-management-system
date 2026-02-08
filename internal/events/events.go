package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"penny-assesment/internal/domain"
)

const (
	AggregateOrder = "order"
	AggregateDrone = "drone"
)

const (
	EventOrderCreated          = "order.created"
	EventOrderReserved         = "order.reserved"
	EventOrderPickedUp         = "order.picked_up"
	EventOrderDelivered        = "order.delivered"
	EventOrderFailed           = "order.failed"
	EventOrderHandoffRequested = "order.handoff_requested"
	EventOrderWithdrawn        = "order.withdrawn"
	EventOrderUpdated          = "order.updated"
	EventDroneBroken           = "drone.broken"
	EventDroneFixed            = "drone.fixed"
)

type Event struct {
	ID            string
	Type          string
	AggregateType string
	AggregateID   string
	Payload       json.RawMessage
	OccurredAt    time.Time
}

func NewEvent(eventType, aggregateType, aggregateID string, payload any, occurredAt time.Time) Event {
	data, _ := json.Marshal(payload)
	return Event{
		ID:            uuid.NewString(),
		Type:          eventType,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		Payload:       data,
		OccurredAt:    occurredAt,
	}
}

func NewOrderEvent(eventType string, order *domain.Order, drone *domain.Drone, occurredAt time.Time) Event {
	payload := map[string]any{
		"order_id":    order.ID,
		"status":      order.Status,
		"user_id":     order.UserID,
		"drone_id":    order.AssignedDroneID,
		"occurred_at": occurredAt,
	}
	if drone != nil {
		payload["drone_status"] = drone.Status
	}
	return NewEvent(eventType, AggregateOrder, order.ID, payload, occurredAt)
}

func NewDroneEvent(eventType string, drone *domain.Drone, occurredAt time.Time) Event {
	payload := map[string]any{
		"drone_id":    drone.ID,
		"status":      drone.Status,
		"occurred_at": occurredAt,
	}
	return NewEvent(eventType, AggregateDrone, drone.ID, payload, occurredAt)
}

