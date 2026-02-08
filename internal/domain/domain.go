package domain

import "time"

const (
	RoleAdmin   = "admin"
	RoleEndUser = "enduser"
	RoleDrone   = "drone"
)

type OrderStatus string

const (
	OrderStatusCreated          OrderStatus = "CREATED"
	OrderStatusReserved         OrderStatus = "RESERVED"
	OrderStatusPickedUp         OrderStatus = "PICKED_UP"
	OrderStatusHandoffRequested OrderStatus = "HANDOFF_REQUESTED"
	OrderStatusDelivered        OrderStatus = "DELIVERED"
	OrderStatusFailed           OrderStatus = "FAILED"
	OrderStatusWithdrawn        OrderStatus = "WITHDRAWN"
)

type DroneStatus string

const (
	DroneStatusActive DroneStatus = "ACTIVE"
	DroneStatusBroken DroneStatus = "BROKEN"
)

type Location struct {
	Lat float64
	Lng float64
}

type Order struct {
	ID               string
	UserID           string
	Origin           Location
	Destination      Location
	Status           OrderStatus
	AssignedDroneID  *string
	HandoffOrigin    *Location
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ReservedAt       *time.Time
	PickedUpAt       *time.Time
	DeliveredAt      *time.Time
	FailedAt         *time.Time
	FailureReason    *string
}

type Drone struct {
	ID              string
	Status          DroneStatus
	LastLocation    *Location
	LastHeartbeatAt *time.Time
	CurrentOrderID  *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func IsTerminal(status OrderStatus) bool {
	switch status {
	case OrderStatusDelivered, OrderStatusFailed, OrderStatusWithdrawn:
		return true
	default:
		return false
	}
}

