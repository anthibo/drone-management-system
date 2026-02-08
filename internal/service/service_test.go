package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"penny-assesment/internal/domain"
	"penny-assesment/internal/events"
)

type memStore struct {
	mu     sync.Mutex
	orders map[string]*domain.Order
	drones map[string]*domain.Drone
}

type memTx struct {
	store *memStore
	closed bool
}

func newMemStore() *memStore {
	return &memStore{
		orders: make(map[string]*domain.Order),
		drones: make(map[string]*domain.Drone),
	}
}

func (m *memStore) BeginTx(ctx context.Context) (Tx, error) {
	m.mu.Lock()
	return &memTx{store: m}, nil
}

func (m *memStore) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	order, ok := m.orders[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copy := *order
	return &copy, nil
}

func (m *memStore) ListOrders(ctx context.Context, filter OrderFilter) ([]*domain.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var orders []*domain.Order
	for _, order := range m.orders {
		if filter.Status != nil && order.Status != *filter.Status {
			continue
		}
		copy := *order
		orders = append(orders, &copy)
	}
	return orders, nil
}

func (m *memStore) CreateOrder(ctx context.Context, order *domain.Order) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.orders[order.ID]; ok {
		return domain.ErrConflict
	}
	copy := *order
	m.orders[order.ID] = &copy
	return nil
}

func (m *memStore) GetDrone(ctx context.Context, id string) (*domain.Drone, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	drone, ok := m.drones[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copy := *drone
	return &copy, nil
}

func (m *memStore) ListDrones(ctx context.Context) ([]*domain.Drone, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var drones []*domain.Drone
	for _, drone := range m.drones {
		copy := *drone
		drones = append(drones, &copy)
	}
	return drones, nil
}

func (t *memTx) Commit(ctx context.Context) error {
	return t.close()
}

func (t *memTx) Rollback(ctx context.Context) error {
	return t.close()
}

func (t *memTx) close() error {
	if t.closed {
		return nil
	}
	t.closed = true
	t.store.mu.Unlock()
	return nil
}

func (t *memTx) GetOrderForUpdate(ctx context.Context, id string) (*domain.Order, error) {
	order, ok := t.store.orders[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copy := *order
	return &copy, nil
}

func (t *memTx) GetDroneForUpdate(ctx context.Context, id string) (*domain.Drone, error) {
	drone, ok := t.store.drones[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copy := *drone
	return &copy, nil
}

func (t *memTx) CreateDrone(ctx context.Context, drone *domain.Drone) error {
	if _, ok := t.store.drones[drone.ID]; ok {
		return domain.ErrConflict
	}
	copy := *drone
	t.store.drones[drone.ID] = &copy
	return nil
}

func (t *memTx) CreateOrder(ctx context.Context, order *domain.Order) error {
	if _, ok := t.store.orders[order.ID]; ok {
		return domain.ErrConflict
	}
	copy := *order
	t.store.orders[order.ID] = &copy
	return nil
}

func (t *memTx) UpdateOrder(ctx context.Context, order *domain.Order) error {
	copy := *order
	t.store.orders[order.ID] = &copy
	return nil
}

func (t *memTx) UpdateDrone(ctx context.Context, drone *domain.Drone) error {
	copy := *drone
	t.store.drones[drone.ID] = &copy
	return nil
}

func (t *memTx) ReserveNextOrder(ctx context.Context, allowed []domain.OrderStatus) (*domain.Order, error) {
	allowedSet := map[domain.OrderStatus]bool{}
	for _, status := range allowed {
		allowedSet[status] = true
	}
	var selected *domain.Order
	for _, order := range t.store.orders {
		if order.AssignedDroneID != nil {
			continue
		}
		if !allowedSet[order.Status] {
			continue
		}
		if selected == nil || order.CreatedAt.Before(selected.CreatedAt) {
			copy := *order
			selected = &copy
		}
	}
	return selected, nil
}

func (t *memTx) EnqueueEvent(ctx context.Context, event events.Event) error {
	return nil
}

func TestComputeETA(t *testing.T) {
	order := &domain.Order{
		ID:          "o1",
		UserID:      "u1",
		Origin:      domain.Location{Lat: 24.7136, Lng: 46.6753},
		Destination: domain.Location{Lat: 24.7743, Lng: 46.7386},
		Status:      domain.OrderStatusCreated,
	}
	eta := ComputeETA(order, nil, 10)
	if eta == nil || *eta <= 0 {
		t.Fatalf("expected ETA to be positive")
	}
	order.Status = domain.OrderStatusDelivered
	eta = ComputeETA(order, nil, 10)
	if eta != nil {
		t.Fatalf("expected ETA to be nil for delivered order")
	}
}

func TestReserveJobConcurrency(t *testing.T) {
	store := newMemStore()
	svc := New(store, 10)
	now := time.Now().UTC()
	order := &domain.Order{
		ID:          "o1",
		UserID:      "u1",
		Origin:      domain.Location{Lat: 1, Lng: 1},
		Destination: domain.Location{Lat: 2, Lng: 2},
		Status:      domain.OrderStatusCreated,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.CreateOrder(context.Background(), order); err != nil {
		t.Fatalf("create order: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	results := make(chan error, 2)
	go func() {
		defer wg.Done()
		_, err := svc.DroneReserveJob(context.Background(), "drone-a")
		results <- err
	}()
	go func() {
		defer wg.Done()
		_, err := svc.DroneReserveJob(context.Background(), "drone-b")
		results <- err
	}()
	wg.Wait()
	close(results)

	var success, noJob int
	for err := range results {
		if err == nil {
			success++
			continue
		}
		if errors.Is(err, domain.ErrNoJob) {
			noJob++
		}
	}
	if success != 1 || noJob != 1 {
		t.Fatalf("expected one success and one no-job, got success=%d noJob=%d", success, noJob)
	}
}

func TestWithdrawClearsDroneAssignment(t *testing.T) {
	store := newMemStore()
	svc := New(store, 10)
	now := time.Now().UTC()
	droneID := "drone-1"
	orderID := "order-1"
	store.drones[droneID] = &domain.Drone{
		ID:             droneID,
		Status:         domain.DroneStatusActive,
		CurrentOrderID: &orderID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	store.orders[orderID] = &domain.Order{
		ID:              orderID,
		UserID:          "user-1",
		Origin:          domain.Location{Lat: 1, Lng: 1},
		Destination:     domain.Location{Lat: 2, Lng: 2},
		Status:          domain.OrderStatusReserved,
		AssignedDroneID: &droneID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	order, err := svc.WithdrawOrder(context.Background(), "user-1", orderID)
	if err != nil {
		t.Fatalf("withdraw: %v", err)
	}
	if order.Status != domain.OrderStatusWithdrawn {
		t.Fatalf("expected withdrawn, got %s", order.Status)
	}
	drone, err := store.GetDrone(context.Background(), droneID)
	if err != nil {
		t.Fatalf("get drone: %v", err)
	}
	if drone.CurrentOrderID != nil {
		t.Fatalf("expected drone current order cleared")
	}
}

func TestMarkBrokenCreatesHandoff(t *testing.T) {
	store := newMemStore()
	svc := New(store, 10)
	now := time.Now().UTC()
	droneID := "drone-1"
	orderID := "order-1"
	loc := &domain.Location{Lat: 5, Lng: 6}
	store.drones[droneID] = &domain.Drone{
		ID:             droneID,
		Status:         domain.DroneStatusActive,
		CurrentOrderID: &orderID,
		LastLocation:   loc,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	store.orders[orderID] = &domain.Order{
		ID:              orderID,
		UserID:          "user-1",
		Origin:          domain.Location{Lat: 1, Lng: 1},
		Destination:     domain.Location{Lat: 2, Lng: 2},
		Status:          domain.OrderStatusPickedUp,
		AssignedDroneID: &droneID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	drone, err := svc.DroneMarkBroken(context.Background(), droneID)
	if err != nil {
		t.Fatalf("mark broken: %v", err)
	}
	if drone.Status != domain.DroneStatusBroken {
		t.Fatalf("expected broken status")
	}
	order, err := store.GetOrder(context.Background(), orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if order.Status != domain.OrderStatusHandoffRequested {
		t.Fatalf("expected handoff requested")
	}
	if order.AssignedDroneID != nil {
		t.Fatalf("expected assigned drone cleared")
	}
	if order.HandoffOrigin == nil || order.HandoffOrigin.Lat != loc.Lat || order.HandoffOrigin.Lng != loc.Lng {
		t.Fatalf("expected handoff origin set")
	}
}
