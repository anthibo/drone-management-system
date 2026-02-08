package service

import (
	"math"

	"penny-assesment/internal/domain"
)

type OrderView struct {
	Order           *domain.Order
	CurrentLocation *domain.Location
	ETASeconds      *int64
}

type DroneStatusView struct {
	Drone        *domain.Drone
	CurrentOrder *OrderView
}

func CurrentLocation(order *domain.Order, drone *domain.Drone) *domain.Location {
	switch order.Status {
	case domain.OrderStatusCreated, domain.OrderStatusReserved:
		loc := order.Origin
		return &loc
	case domain.OrderStatusPickedUp:
		if drone != nil && drone.LastLocation != nil {
			loc := *drone.LastLocation
			return &loc
		}
	case domain.OrderStatusHandoffRequested:
		if order.HandoffOrigin != nil {
			loc := *order.HandoffOrigin
			return &loc
		}
	}
	return nil
}

func ComputeETA(order *domain.Order, drone *domain.Drone, speedMPS float64) *int64 {
	if domain.IsTerminal(order.Status) {
		return nil
	}
	if speedMPS <= 0 {
		return nil
	}
	var from domain.Location
	switch order.Status {
	case domain.OrderStatusCreated, domain.OrderStatusReserved:
		from = order.Origin
	case domain.OrderStatusPickedUp:
		if drone == nil || drone.LastLocation == nil {
			return nil
		}
		from = *drone.LastLocation
	case domain.OrderStatusHandoffRequested:
		if order.HandoffOrigin == nil {
			return nil
		}
		from = *order.HandoffOrigin
	default:
		return nil
	}
	dist := haversineMeters(from, order.Destination)
	seconds := int64(dist / speedMPS)
	if seconds < 0 {
		seconds = 0
	}
	return &seconds
}

func haversineMeters(a, b domain.Location) float64 {
	const earthRadius = 6371000.0
	lat1 := degreesToRadians(a.Lat)
	lat2 := degreesToRadians(b.Lat)
	dLat := degreesToRadians(b.Lat - a.Lat)
	dLng := degreesToRadians(b.Lng - a.Lng)

	sinLat := math.Sin(dLat / 2)
	sinLng := math.Sin(dLng / 2)

	h := sinLat*sinLat + math.Cos(lat1)*math.Cos(lat2)*sinLng*sinLng
	return 2 * earthRadius * math.Asin(math.Sqrt(h))
}

func degreesToRadians(deg float64) float64 {
	return deg * math.Pi / 180
}
