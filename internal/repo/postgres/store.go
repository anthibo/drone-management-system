package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"penny-assesment/internal/domain"
	"penny-assesment/internal/events"
	"penny-assesment/internal/service"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) BeginTx(ctx context.Context) (service.Tx, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	return &Tx{tx: tx}, nil
}

func (s *Store) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	row := s.pool.QueryRow(ctx, orderSelectByIDSQL, id)
	return scanOrder(row)
}

func (s *Store) ListOrders(ctx context.Context, filter service.OrderFilter) ([]*domain.Order, error) {
	status := sql.NullString{}
	if filter.Status != nil {
		status = sql.NullString{String: string(*filter.Status), Valid: true}
	}
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, orderListSQL, status, limit, filter.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return orders, nil
}

func (s *Store) CreateOrder(ctx context.Context, order *domain.Order) error {
	_, err := s.pool.Exec(ctx, orderInsertSQL,
		order.ID,
		order.UserID,
		order.Origin.Lat,
		order.Origin.Lng,
		order.Destination.Lat,
		order.Destination.Lng,
		order.Status,
		nullString(order.AssignedDroneID),
		optionalLocationLat(order.HandoffOrigin),
		optionalLocationLng(order.HandoffOrigin),
		order.CreatedAt,
		order.UpdatedAt,
		nullTime(order.ReservedAt),
		nullTime(order.PickedUpAt),
		nullTime(order.DeliveredAt),
		nullTime(order.FailedAt),
		nullString(order.FailureReason),
	)
	return err
}

func (s *Store) GetDrone(ctx context.Context, id string) (*domain.Drone, error) {
	row := s.pool.QueryRow(ctx, droneSelectByIDSQL, id)
	return scanDrone(row)
}

func (s *Store) ListDrones(ctx context.Context) ([]*domain.Drone, error) {
	rows, err := s.pool.Query(ctx, droneListSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var drones []*domain.Drone
	for rows.Next() {
		drone, err := scanDrone(rows)
		if err != nil {
			return nil, err
		}
		drones = append(drones, drone)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return drones, nil
}

type Tx struct {
	tx pgx.Tx
}

func (t *Tx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *Tx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

func (t *Tx) GetOrderForUpdate(ctx context.Context, id string) (*domain.Order, error) {
	row := t.tx.QueryRow(ctx, orderSelectByIDForUpdateSQL, id)
	return scanOrder(row)
}

func (t *Tx) GetDroneForUpdate(ctx context.Context, id string) (*domain.Drone, error) {
	row := t.tx.QueryRow(ctx, droneSelectByIDForUpdateSQL, id)
	return scanDrone(row)
}

func (t *Tx) CreateDrone(ctx context.Context, drone *domain.Drone) error {
	_, err := t.tx.Exec(ctx, droneInsertSQL,
		drone.ID,
		drone.Status,
		nullLocationLat(drone.LastLocation),
		nullLocationLng(drone.LastLocation),
		nullTime(drone.LastHeartbeatAt),
		nullString(drone.CurrentOrderID),
		drone.CreatedAt,
		drone.UpdatedAt,
	)
	return err
}

func (t *Tx) CreateOrder(ctx context.Context, order *domain.Order) error {
	_, err := t.tx.Exec(ctx, orderInsertSQL,
		order.ID,
		order.UserID,
		order.Origin.Lat,
		order.Origin.Lng,
		order.Destination.Lat,
		order.Destination.Lng,
		order.Status,
		nullString(order.AssignedDroneID),
		optionalLocationLat(order.HandoffOrigin),
		optionalLocationLng(order.HandoffOrigin),
		order.CreatedAt,
		order.UpdatedAt,
		nullTime(order.ReservedAt),
		nullTime(order.PickedUpAt),
		nullTime(order.DeliveredAt),
		nullTime(order.FailedAt),
		nullString(order.FailureReason),
	)
	return err
}

func (t *Tx) UpdateOrder(ctx context.Context, order *domain.Order) error {
	_, err := t.tx.Exec(ctx, orderUpdateSQL,
		order.UserID,
		order.Origin.Lat,
		order.Origin.Lng,
		order.Destination.Lat,
		order.Destination.Lng,
		order.Status,
		nullString(order.AssignedDroneID),
		optionalLocationLat(order.HandoffOrigin),
		optionalLocationLng(order.HandoffOrigin),
		order.UpdatedAt,
		nullTime(order.ReservedAt),
		nullTime(order.PickedUpAt),
		nullTime(order.DeliveredAt),
		nullTime(order.FailedAt),
		nullString(order.FailureReason),
		order.ID,
	)
	return err
}

func (t *Tx) UpdateDrone(ctx context.Context, drone *domain.Drone) error {
	_, err := t.tx.Exec(ctx, droneUpdateSQL,
		drone.Status,
		nullLocationLat(drone.LastLocation),
		nullLocationLng(drone.LastLocation),
		nullTime(drone.LastHeartbeatAt),
		nullString(drone.CurrentOrderID),
		drone.UpdatedAt,
		drone.ID,
	)
	return err
}

func (t *Tx) ReserveNextOrder(ctx context.Context, allowed []domain.OrderStatus) (*domain.Order, error) {
	if len(allowed) == 0 {
		return nil, nil
	}
	allowedVals := make([]string, 0, len(allowed))
	for _, status := range allowed {
		allowedVals = append(allowedVals, string(status))
	}
	rows, err := t.tx.Query(ctx, orderReserveSQL, allowedVals)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		if rows.Err() != nil {
			return nil, rows.Err()
		}
		return nil, nil
	}
	order, err := scanOrder(rows)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (t *Tx) EnqueueEvent(ctx context.Context, event events.Event) error {
	_, err := t.tx.Exec(ctx, outboxInsertSQL,
		event.ID,
		event.Type,
		event.AggregateType,
		event.AggregateID,
		event.Payload,
		event.OccurredAt,
	)
	return err
}

func scanOrder(row pgx.Row) (*domain.Order, error) {
	var (
		assignedDroneID sql.NullString
		handoffLat      sql.NullFloat64
		handoffLng      sql.NullFloat64
		reservedAt      sql.NullTime
		pickedUpAt      sql.NullTime
		deliveredAt     sql.NullTime
		failedAt        sql.NullTime
		failureReason   sql.NullString
	)
	order := &domain.Order{}
	err := row.Scan(
		&order.ID,
		&order.UserID,
		&order.Origin.Lat,
		&order.Origin.Lng,
		&order.Destination.Lat,
		&order.Destination.Lng,
		&order.Status,
		&assignedDroneID,
		&handoffLat,
		&handoffLng,
		&order.CreatedAt,
		&order.UpdatedAt,
		&reservedAt,
		&pickedUpAt,
		&deliveredAt,
		&failedAt,
		&failureReason,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if assignedDroneID.Valid {
		order.AssignedDroneID = &assignedDroneID.String
	}
	if handoffLat.Valid && handoffLng.Valid {
		order.HandoffOrigin = &domain.Location{Lat: handoffLat.Float64, Lng: handoffLng.Float64}
	}
	if reservedAt.Valid {
		order.ReservedAt = &reservedAt.Time
	}
	if pickedUpAt.Valid {
		order.PickedUpAt = &pickedUpAt.Time
	}
	if deliveredAt.Valid {
		order.DeliveredAt = &deliveredAt.Time
	}
	if failedAt.Valid {
		order.FailedAt = &failedAt.Time
	}
	if failureReason.Valid {
		order.FailureReason = &failureReason.String
	}
	return order, nil
}

func scanDrone(row pgx.Row) (*domain.Drone, error) {
	var (
		lastLat         sql.NullFloat64
		lastLng         sql.NullFloat64
		lastHeartbeatAt sql.NullTime
		currentOrderID  sql.NullString
	)
	drone := &domain.Drone{}
	err := row.Scan(
		&drone.ID,
		&drone.Status,
		&lastLat,
		&lastLng,
		&lastHeartbeatAt,
		&currentOrderID,
		&drone.CreatedAt,
		&drone.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if lastLat.Valid && lastLng.Valid {
		drone.LastLocation = &domain.Location{Lat: lastLat.Float64, Lng: lastLng.Float64}
	}
	if lastHeartbeatAt.Valid {
		drone.LastHeartbeatAt = &lastHeartbeatAt.Time
	}
	if currentOrderID.Valid {
		drone.CurrentOrderID = &currentOrderID.String
	}
	return drone, nil
}

func nullString(v *string) sql.NullString {
	if v == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *v, Valid: true}
}

func nullTime(v *time.Time) sql.NullTime {
	if v == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *v, Valid: true}
}

func nullLocationLat(loc *domain.Location) sql.NullFloat64 {
	if loc == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: loc.Lat, Valid: true}
}

func nullLocationLng(loc *domain.Location) sql.NullFloat64 {
	if loc == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: loc.Lng, Valid: true}
}

func optionalLocationLat(loc *domain.Location) sql.NullFloat64 {
	return nullLocationLat(loc)
}

func optionalLocationLng(loc *domain.Location) sql.NullFloat64 {
	return nullLocationLng(loc)
}
