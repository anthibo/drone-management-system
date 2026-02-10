package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"penny-assesment/internal/domain"
	"penny-assesment/internal/events"
)

type Store interface {
	BeginTx(ctx context.Context) (Tx, error)
	GetOrder(ctx context.Context, id string) (*domain.Order, error)
	ListOrders(ctx context.Context, filter OrderFilter) ([]*domain.Order, error)
	CreateOrder(ctx context.Context, order *domain.Order) error
	GetDrone(ctx context.Context, id string) (*domain.Drone, error)
	ListDrones(ctx context.Context) ([]*domain.Drone, error)
}

type Tx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
	GetOrderForUpdate(ctx context.Context, id string) (*domain.Order, error)
	GetDroneForUpdate(ctx context.Context, id string) (*domain.Drone, error)
	CreateDrone(ctx context.Context, drone *domain.Drone) error
	CreateOrder(ctx context.Context, order *domain.Order) error
	UpdateOrder(ctx context.Context, order *domain.Order) error
	UpdateDrone(ctx context.Context, drone *domain.Drone) error
	ReserveNextOrder(ctx context.Context, allowed []domain.OrderStatus) (*domain.Order, error)
	EnqueueEvent(ctx context.Context, event events.Event) error
}

type OrderFilter struct {
	Status *domain.OrderStatus
	Limit  int
	Offset int
}

type Service struct {
	store Store
	now   func() time.Time
	speed float64
}

func New(store Store, speedMPS float64) *Service {
	return &Service{store: store, now: func() time.Time { return time.Now().UTC() }, speed: speedMPS}
}

func (s *Service) SubmitOrder(ctx context.Context, userID string, origin, dest domain.Location) (*domain.Order, error) {
	if err := domain.ValidateLocation(origin); err != nil {
		return nil, fmt.Errorf("origin: %w", domain.ErrInvalid)
	}
	if err := domain.ValidateLocation(dest); err != nil {
		return nil, fmt.Errorf("destination: %w", domain.ErrInvalid)
	}
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	now := s.now()
	order := &domain.Order{
		ID:          newOrderID(),
		UserID:      userID,
		Origin:      origin,
		Destination: dest,
		Status:      domain.OrderStatusCreated,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := tx.CreateOrder(ctx, order); err != nil {
		return nil, err
	}
	if err := tx.EnqueueEvent(ctx, events.NewOrderEvent(events.EventOrderCreated, order, nil, now)); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *Service) WithdrawOrder(ctx context.Context, userID, orderID string) (*domain.Order, error) {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	order, err := tx.GetOrderForUpdate(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, domain.ErrForbidden
	}
	if order.Status != domain.OrderStatusCreated && order.Status != domain.OrderStatusReserved {
		return nil, domain.ErrPrecondition
	}
	if order.Status == domain.OrderStatusReserved && order.AssignedDroneID != nil {
		drone, err := tx.GetDroneForUpdate(ctx, *order.AssignedDroneID)
		if err != nil {
			return nil, err
		}
		drone.CurrentOrderID = nil
		drone.UpdatedAt = s.now()
		if err := tx.UpdateDrone(ctx, drone); err != nil {
			return nil, err
		}
	}
	now := s.now()
	order.Status = domain.OrderStatusWithdrawn
	order.AssignedDroneID = nil
	order.UpdatedAt = now
	if err := tx.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}
	if err := tx.EnqueueEvent(ctx, events.NewOrderEvent(events.EventOrderWithdrawn, order, nil, now)); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *Service) GetOrderView(ctx context.Context, requesterID, role, orderID string) (*OrderView, error) {
	order, err := s.store.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if role != domain.RoleAdmin && order.UserID != requesterID {
		return nil, domain.ErrForbidden
	}
	return s.buildOrderView(ctx, order)
}

func (s *Service) AdminListOrders(ctx context.Context, filter OrderFilter) ([]*OrderView, error) {
	orders, err := s.store.ListOrders(ctx, filter)
	if err != nil {
		return nil, err
	}
	views := make([]*OrderView, 0, len(orders))
	for _, order := range orders {
		view, err := s.buildOrderView(ctx, order)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	sort.Slice(views, func(i, j int) bool {
		return views[i].Order.CreatedAt.Before(views[j].Order.CreatedAt)
	})
	return views, nil
}

func (s *Service) AdminUpdateOrder(ctx context.Context, orderID string, origin, dest *domain.Location) (*domain.Order, error) {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	order, err := tx.GetOrderForUpdate(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if domain.IsTerminal(order.Status) {
		return nil, domain.ErrPrecondition
	}
	if origin != nil {
		if err := domain.ValidateLocation(*origin); err != nil {
			return nil, domain.ErrInvalid
		}
		order.Origin = *origin
	}
	if dest != nil {
		if err := domain.ValidateLocation(*dest); err != nil {
			return nil, domain.ErrInvalid
		}
		order.Destination = *dest
	}
	order.UpdatedAt = s.now()
	if err := tx.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}
	if err := tx.EnqueueEvent(ctx, events.NewOrderEvent(events.EventOrderUpdated, order, nil, s.now())); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *Service) DroneReserveJob(ctx context.Context, droneID string) (*domain.Order, error) {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	drone, err := getOrCreateDrone(ctx, tx, droneID, s.now())
	if err != nil {
		return nil, err
	}
	if drone.Status == domain.DroneStatusBroken {
		return nil, domain.ErrPrecondition
	}
	if drone.CurrentOrderID != nil {
		return nil, domain.ErrConflict
	}
	order, err := tx.ReserveNextOrder(ctx, []domain.OrderStatus{domain.OrderStatusCreated, domain.OrderStatusHandoffRequested})
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, domain.ErrNoJob
	}
	now := s.now()
	order.Status = domain.OrderStatusReserved
	order.AssignedDroneID = &drone.ID
	order.ReservedAt = &now
	order.UpdatedAt = now
	if err := tx.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}
	drone.CurrentOrderID = &order.ID
	drone.UpdatedAt = now
	if err := tx.UpdateDrone(ctx, drone); err != nil {
		return nil, err
	}
	if err := tx.EnqueueEvent(ctx, events.NewOrderEvent(events.EventOrderReserved, order, drone, now)); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *Service) DronePickup(ctx context.Context, droneID, orderID string) (*domain.Order, error) {
	return s.updateOrderForDrone(ctx, droneID, orderID, events.EventOrderPickedUp, func(order *domain.Order) error {
		if order.Status != domain.OrderStatusReserved && order.Status != domain.OrderStatusHandoffRequested {
			return domain.ErrPrecondition
		}
		now := s.now()
		order.Status = domain.OrderStatusPickedUp
		order.PickedUpAt = &now
		order.UpdatedAt = now
		return nil
	})
}

func (s *Service) DroneDeliver(ctx context.Context, droneID, orderID string) (*domain.Order, error) {
	return s.completeOrderForDrone(ctx, droneID, orderID, domain.OrderStatusDelivered, "")
}

func (s *Service) DroneFail(ctx context.Context, droneID, orderID, reason string) (*domain.Order, error) {
	if reason == "" {
		return nil, domain.ErrInvalid
	}
	return s.completeOrderForDrone(ctx, droneID, orderID, domain.OrderStatusFailed, reason)
}

func (s *Service) DroneMarkBroken(ctx context.Context, droneID string) (*domain.Drone, error) {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	drone, err := getOrCreateDrone(ctx, tx, droneID, s.now())
	if err != nil {
		return nil, err
	}
	now := s.now()
	drone.Status = domain.DroneStatusBroken
	if drone.CurrentOrderID != nil {
		order, err := tx.GetOrderForUpdate(ctx, *drone.CurrentOrderID)
		if err != nil {
			return nil, err
		}

		// Only create a handoff job if the package is actually in-flight.
		// If the order is merely RESERVED (not picked up yet), requeue it back to CREATED.
		switch order.Status {
		case domain.OrderStatusPickedUp:
			order.Status = domain.OrderStatusHandoffRequested
			order.AssignedDroneID = nil
			if drone.LastLocation != nil {
				loc := *drone.LastLocation
				order.HandoffOrigin = &loc
			}
			order.UpdatedAt = now
			if err := tx.UpdateOrder(ctx, order); err != nil {
				return nil, err
			}
			drone.CurrentOrderID = nil
			if err := tx.EnqueueEvent(ctx, events.NewOrderEvent(events.EventOrderHandoffRequested, order, drone, now)); err != nil {
				return nil, err
			}
		case domain.OrderStatusReserved:
			order.Status = domain.OrderStatusCreated
			order.AssignedDroneID = nil
			order.ReservedAt = nil
			order.HandoffOrigin = nil
			order.UpdatedAt = now
			if err := tx.UpdateOrder(ctx, order); err != nil {
				return nil, err
			}
			drone.CurrentOrderID = nil
			if err := tx.EnqueueEvent(ctx, events.NewOrderEvent(events.EventOrderUpdated, order, drone, now)); err != nil {
				return nil, err
			}
		default:
			// For any other state, don't mutate the order; still mark drone broken.
			drone.CurrentOrderID = nil
		}
	}
	drone.UpdatedAt = now
	if err := tx.UpdateDrone(ctx, drone); err != nil {
		return nil, err
	}
	if err := tx.EnqueueEvent(ctx, events.NewDroneEvent(events.EventDroneBroken, drone, now)); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return drone, nil
}

func (s *Service) DroneMarkFixed(ctx context.Context, droneID string) (*domain.Drone, error) {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	drone, err := getOrCreateDrone(ctx, tx, droneID, s.now())
	if err != nil {
		return nil, err
	}
	now := s.now()
	drone.Status = domain.DroneStatusActive
	drone.UpdatedAt = now
	if err := tx.UpdateDrone(ctx, drone); err != nil {
		return nil, err
	}
	if err := tx.EnqueueEvent(ctx, events.NewDroneEvent(events.EventDroneFixed, drone, now)); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return drone, nil
}

func (s *Service) DroneHeartbeat(ctx context.Context, droneID string, loc domain.Location) (*DroneStatusView, error) {
	if err := domain.ValidateLocation(loc); err != nil {
		return nil, domain.ErrInvalid
	}
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	drone, err := getOrCreateDrone(ctx, tx, droneID, s.now())
	if err != nil {
		return nil, err
	}
	now := s.now()
	drone.LastLocation = &loc
	drone.LastHeartbeatAt = &now
	drone.UpdatedAt = now
	if err := tx.UpdateDrone(ctx, drone); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	var orderView *OrderView
	if drone.CurrentOrderID != nil {
		order, err := s.store.GetOrder(ctx, *drone.CurrentOrderID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		if order != nil {
			orderView, err = s.buildOrderView(ctx, order)
			if err != nil {
				return nil, err
			}
		}
	}
	return &DroneStatusView{Drone: drone, CurrentOrder: orderView}, nil
}

func (s *Service) DroneCurrentOrder(ctx context.Context, droneID string) (*OrderView, error) {
	drone, err := s.store.GetDrone(ctx, droneID)
	if err != nil {
		return nil, err
	}
	if drone.CurrentOrderID == nil {
		return nil, domain.ErrNotFound
	}
	order, err := s.store.GetOrder(ctx, *drone.CurrentOrderID)
	if err != nil {
		return nil, err
	}
	return s.buildOrderView(ctx, order)
}

func (s *Service) AdminListDrones(ctx context.Context) ([]*domain.Drone, error) {
	return s.store.ListDrones(ctx)
}

func (s *Service) AdminMarkDroneBroken(ctx context.Context, droneID string) (*domain.Drone, error) {
	return s.DroneMarkBroken(ctx, droneID)
}

func (s *Service) AdminMarkDroneFixed(ctx context.Context, droneID string) (*domain.Drone, error) {
	return s.DroneMarkFixed(ctx, droneID)
}

func (s *Service) updateOrderForDrone(ctx context.Context, droneID, orderID string, eventType string, fn func(order *domain.Order) error) (*domain.Order, error) {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	order, err := tx.GetOrderForUpdate(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.AssignedDroneID == nil || *order.AssignedDroneID != droneID {
		return nil, domain.ErrForbidden
	}
	if err := fn(order); err != nil {
		return nil, err
	}
	if err := tx.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}
	if err := tx.EnqueueEvent(ctx, events.NewOrderEvent(eventType, order, nil, s.now())); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *Service) completeOrderForDrone(ctx context.Context, droneID, orderID string, status domain.OrderStatus, reason string) (*domain.Order, error) {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	order, err := tx.GetOrderForUpdate(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.AssignedDroneID == nil || *order.AssignedDroneID != droneID {
		return nil, domain.ErrForbidden
	}
	if order.Status != domain.OrderStatusPickedUp {
		return nil, domain.ErrPrecondition
	}
	now := s.now()
	order.Status = status
	order.UpdatedAt = now
	if status == domain.OrderStatusDelivered {
		order.DeliveredAt = &now
	} else {
		order.FailedAt = &now
		if reason != "" {
			order.FailureReason = &reason
		}
	}
	if err := tx.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}
	drone, err := tx.GetDroneForUpdate(ctx, droneID)
	if err != nil {
		return nil, err
	}
	drone.CurrentOrderID = nil
	drone.UpdatedAt = now
	if err := tx.UpdateDrone(ctx, drone); err != nil {
		return nil, err
	}
	eventType := events.EventOrderDelivered
	if status == domain.OrderStatusFailed {
		eventType = events.EventOrderFailed
	}
	if err := tx.EnqueueEvent(ctx, events.NewOrderEvent(eventType, order, drone, now)); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *Service) buildOrderView(ctx context.Context, order *domain.Order) (*OrderView, error) {
	var drone *domain.Drone
	if order.AssignedDroneID != nil {
		var err error
		drone, err = s.store.GetDrone(ctx, *order.AssignedDroneID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
	}
	eta := ComputeETA(order, drone, s.speed)
	loc := CurrentLocation(order, drone)
	return &OrderView{Order: order, CurrentLocation: loc, ETASeconds: eta}, nil
}

func getOrCreateDrone(ctx context.Context, tx Tx, droneID string, now time.Time) (*domain.Drone, error) {
	drone, err := tx.GetDroneForUpdate(ctx, droneID)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		drone = &domain.Drone{
			ID:        droneID,
			Status:    domain.DroneStatusActive,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := tx.CreateDrone(ctx, drone); err != nil {
			return nil, err
		}
	}
	return drone, nil
}

func newOrderID() string {
	return uuidFunc()
}

var uuidFunc = func() string { return "" }
