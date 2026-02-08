package transport

import (
	"time"

	"penny-assesment/internal/domain"
	"penny-assesment/internal/service"
)

type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type OrderResponse struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	Origin          Location   `json:"origin"`
	Destination     Location   `json:"destination"`
	Status          string     `json:"status"`
	AssignedDroneID *string    `json:"assigned_drone_id,omitempty"`
	HandoffOrigin   *Location  `json:"handoff_origin,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ReservedAt      *time.Time `json:"reserved_at,omitempty"`
	PickedUpAt      *time.Time `json:"picked_up_at,omitempty"`
	DeliveredAt     *time.Time `json:"delivered_at,omitempty"`
	FailedAt        *time.Time `json:"failed_at,omitempty"`
	FailureReason   *string    `json:"failure_reason,omitempty"`
}

type OrderViewResponse struct {
	Order           OrderResponse `json:"order"`
	CurrentLocation *Location     `json:"current_location,omitempty"`
	ETASeconds      *int64        `json:"eta_seconds,omitempty"`
}

type DroneResponse struct {
	ID              string     `json:"id"`
	Status          string     `json:"status"`
	LastLocation    *Location  `json:"last_location,omitempty"`
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at,omitempty"`
	CurrentOrderID  *string    `json:"current_order_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type DroneStatusResponse struct {
	Drone        DroneResponse     `json:"drone"`
	CurrentOrder *OrderViewResponse `json:"current_order,omitempty"`
}

func FromOrder(order *domain.Order) OrderResponse {
	resp := OrderResponse{
		ID:              order.ID,
		UserID:          order.UserID,
		Origin:          Location{Lat: order.Origin.Lat, Lng: order.Origin.Lng},
		Destination:     Location{Lat: order.Destination.Lat, Lng: order.Destination.Lng},
		Status:          string(order.Status),
		AssignedDroneID: order.AssignedDroneID,
		CreatedAt:       order.CreatedAt,
		UpdatedAt:       order.UpdatedAt,
		ReservedAt:      order.ReservedAt,
		PickedUpAt:      order.PickedUpAt,
		DeliveredAt:     order.DeliveredAt,
		FailedAt:        order.FailedAt,
		FailureReason:   order.FailureReason,
	}
	if order.HandoffOrigin != nil {
		resp.HandoffOrigin = &Location{Lat: order.HandoffOrigin.Lat, Lng: order.HandoffOrigin.Lng}
	}
	return resp
}

func FromOrderView(view *service.OrderView) OrderViewResponse {
	resp := OrderViewResponse{
		Order:      FromOrder(view.Order),
		ETASeconds: view.ETASeconds,
	}
	if view.CurrentLocation != nil {
		resp.CurrentLocation = &Location{Lat: view.CurrentLocation.Lat, Lng: view.CurrentLocation.Lng}
	}
	return resp
}

func FromDrone(drone *domain.Drone) DroneResponse {
	resp := DroneResponse{
		ID:              drone.ID,
		Status:          string(drone.Status),
		CurrentOrderID:  drone.CurrentOrderID,
		CreatedAt:       drone.CreatedAt,
		UpdatedAt:       drone.UpdatedAt,
		LastHeartbeatAt: drone.LastHeartbeatAt,
	}
	if drone.LastLocation != nil {
		resp.LastLocation = &Location{Lat: drone.LastLocation.Lat, Lng: drone.LastLocation.Lng}
	}
	return resp
}

func FromDroneStatus(view *service.DroneStatusView) DroneStatusResponse {
	resp := DroneStatusResponse{
		Drone: FromDrone(view.Drone),
	}
	if view.CurrentOrder != nil {
		orderView := FromOrderView(view.CurrentOrder)
		resp.CurrentOrder = &orderView
	}
	return resp
}
