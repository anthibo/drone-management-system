package grpcapi

import "penny-assesment/internal/transport"

type TokenRequest struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

type TokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

type SubmitOrderRequest struct {
	Origin      transport.Location `json:"origin"`
	Destination transport.Location `json:"destination"`
}

type OrderIDRequest struct {
	OrderID string `json:"order_id"`
}

type FailOrderRequest struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

type HeartbeatRequest struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type ListOrdersRequest struct {
	Status string `json:"status"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type UpdateOrderRequest struct {
	OrderID     string               `json:"order_id"`
	Origin      *transport.Location `json:"origin"`
	Destination *transport.Location `json:"destination"`
}

type DroneIDRequest struct {
	DroneID string `json:"drone_id"`
}

