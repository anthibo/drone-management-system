package thriftapi

import (
	"context"
	"errors"
	"time"

	"github.com/apache/thrift/lib/go/thrift"

	"penny-assesment/internal/auth"
	"penny-assesment/internal/domain"
	"penny-assesment/internal/service"
)

type Processor struct {
	svc          *service.Service
	auth         *auth.Authenticator
	processorMap map[string]thrift.TProcessorFunction
}

type handlerFunc func(ctx context.Context, seqID int32, in, out thrift.TProtocol) (bool, thrift.TException)

type processorFunc struct {
	fn handlerFunc
}

func (p processorFunc) Process(ctx context.Context, seqID int32, in, out thrift.TProtocol) (bool, thrift.TException) {
	return p.fn(ctx, seqID, in, out)
}

func NewProcessor(svc *service.Service, authenticator *auth.Authenticator) *Processor {
	p := &Processor{svc: svc, auth: authenticator}
	p.processorMap = map[string]thrift.TProcessorFunction{
		"IssueToken":      processorFunc{fn: p.handleIssueToken},
		"SubmitOrder":     processorFunc{fn: p.handleSubmitOrder},
		"WithdrawOrder":   processorFunc{fn: p.handleWithdrawOrder},
		"GetOrder":        processorFunc{fn: p.handleGetOrder},
		"ReserveJob":      processorFunc{fn: p.handleReserveJob},
		"PickupOrder":     processorFunc{fn: p.handlePickupOrder},
		"DeliverOrder":    processorFunc{fn: p.handleDeliverOrder},
		"FailOrder":       processorFunc{fn: p.handleFailOrder},
		"MarkBroken":      processorFunc{fn: p.handleMarkBroken},
		"Heartbeat":       processorFunc{fn: p.handleHeartbeat},
		"CurrentOrder":    processorFunc{fn: p.handleCurrentOrder},
		"ListOrders":      processorFunc{fn: p.handleAdminListOrders},
		"UpdateOrder":     processorFunc{fn: p.handleAdminUpdateOrder},
		"ListDrones":      processorFunc{fn: p.handleAdminListDrones},
		"MarkDroneBroken": processorFunc{fn: p.handleAdminMarkDroneBroken},
		"MarkDroneFixed":  processorFunc{fn: p.handleAdminMarkDroneFixed},
	}
	return p
}

func (p *Processor) ProcessorMap() map[string]thrift.TProcessorFunction {
	return p.processorMap
}

func (p *Processor) AddToProcessorMap(name string, processor thrift.TProcessorFunction) {
	p.processorMap[name] = processor
}

func (p *Processor) Process(ctx context.Context, in, out thrift.TProtocol) (bool, thrift.TException) {
	name, messageType, seqID, err := in.ReadMessageBegin(ctx)
	if err != nil {
		return false, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error())
	}
	if messageType != thrift.CALL && messageType != thrift.ONEWAY {
		return p.writeException(ctx, out, name, seqID, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, "invalid message type"))
	}
	processor, ok := p.processorMap[name]
	if !ok {
		_ = in.Skip(ctx, thrift.STRUCT)
		_ = in.ReadMessageEnd(ctx)
		return p.writeException(ctx, out, name, seqID, thrift.NewTApplicationException(thrift.UNKNOWN_METHOD, "unknown method"))
	}
	return processor.Process(ctx, seqID, in, out)
}

func (p *Processor) handleIssueToken(ctx context.Context, seqID int32, in, out thrift.TProtocol) (bool, thrift.TException) {
	name, role, err := readTokenRequest(ctx, in)
	if err != nil {
		return p.writeException(ctx, out, "IssueToken", seqID, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error()))
	}
	if name == "" || !domain.ValidateRole(role) {
		return p.writeException(ctx, out, "IssueToken", seqID, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, "invalid request"))
	}
	jwt, exp, err := p.auth.IssueToken(name, role)
	if err != nil {
		return p.writeException(ctx, out, "IssueToken", seqID, thrift.NewTApplicationException(thrift.INTERNAL_ERROR, "token error"))
	}
	return p.writeReply(ctx, out, "IssueToken", seqID, func(out thrift.TProtocol) error {
		if err := out.WriteFieldBegin(ctx, "success", thrift.STRUCT, 0); err != nil {
			return err
		}
		return writeTokenResponse(ctx, out, jwt, exp)
	})
}

func (p *Processor) handleSubmitOrder(ctx context.Context, seqID int32, in, out thrift.TProtocol) (bool, thrift.TException) {
	authToken, origin, dest, err := readSubmitOrderRequest(ctx, in)
	if err != nil {
		return p.writeException(ctx, out, "SubmitOrder", seqID, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error()))
	}
	claims, appErr := p.authorize(authToken, domain.RoleEndUser)
	if appErr != nil {
		return p.writeException(ctx, out, "SubmitOrder", seqID, appErr)
	}
	order, err := p.svc.SubmitOrder(ctx, claims.Subject, origin, dest)
	if err != nil {
		return p.writeException(ctx, out, "SubmitOrder", seqID, mapError(err))
	}
	return p.writeReply(ctx, out, "SubmitOrder", seqID, func(out thrift.TProtocol) error {
		if err := out.WriteFieldBegin(ctx, "success", thrift.STRUCT, 0); err != nil {
			return err
		}
		return writeOrder(ctx, out, order)
	})
}

func (p *Processor) handleWithdrawOrder(ctx context.Context, seqID int32, in, out thrift.TProtocol) (bool, thrift.TException) {
	authToken, orderID, err := readOrderIDRequest(ctx, in)
	if err != nil {
		return p.writeException(ctx, out, "WithdrawOrder", seqID, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error()))
	}
	claims, appErr := p.authorize(authToken, domain.RoleEndUser)
	if appErr != nil {
		return p.writeException(ctx, out, "WithdrawOrder", seqID, appErr)
	}
	order, err := p.svc.WithdrawOrder(ctx, claims.Subject, orderID)
	if err != nil {
		return p.writeException(ctx, out, "WithdrawOrder", seqID, mapError(err))
	}
	return p.writeReply(ctx, out, "WithdrawOrder", seqID, func(out thrift.TProtocol) error {
		if err := out.WriteFieldBegin(ctx, "success", thrift.STRUCT, 0); err != nil {
			return err
		}
		return writeOrder(ctx, out, order)
	})
}

func (p *Processor) handleGetOrder(ctx context.Context, seqID int32, in, out thrift.TProtocol) (bool, thrift.TException) {
	authToken, orderID, err := readOrderIDRequest(ctx, in)
	if err != nil {
		return p.writeException(ctx, out, "GetOrder", seqID, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error()))
	}
	claims, appErr := p.authorizeAny(authToken)
	if appErr != nil {
		return p.writeException(ctx, out, "GetOrder", seqID, appErr)
	}
	view, err := p.svc.GetOrderView(ctx, claims.Subject, claims.Role, orderID)
	if err != nil {
		return p.writeException(ctx, out, "GetOrder", seqID, mapError(err))
	}
	return p.writeReply(ctx, out, "GetOrder", seqID, func(out thrift.TProtocol) error {
		if err := out.WriteFieldBegin(ctx, "success", thrift.STRUCT, 0); err != nil {
			return err
		}
		return writeOrderView(ctx, out, view)
	})
}

func (p *Processor) handleReserveJob(ctx context.Context, seqID int32, in, out thrift.TProtocol) (bool, thrift.TException) {
	authToken, err := readAuthRequest(ctx, in)
	if err != nil {
		return p.writeException(ctx, out, "ReserveJob", seqID, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error()))
	}
	claims, appErr := p.authorize(authToken, domain.RoleDrone)
	if appErr != nil {
		return p.writeException(ctx, out, "ReserveJob", seqID, appErr)
	}
	order, err := p.svc.DroneReserveJob(ctx, claims.Subject)
	if err != nil {
		return p.writeException(ctx, out, "ReserveJob", seqID, mapError(err))
	}
	return p.writeReply(ctx, out, "ReserveJob", seqID, func(out thrift.TProtocol) error {
		if err := out.WriteFieldBegin(ctx, "success", thrift.STRUCT, 0); err != nil {
			return err
		}
		return writeOrder(ctx, out, order)
	})
}

func (p *Processor) handlePickupOrder(ctx context.Context, seqID int32, in, out thrift.TProtocol) (bool, thrift.TException) {
	authToken, orderID, err := readOrderIDRequest(ctx, in)
	if err != nil {
		return p.writeException(ctx, out, "PickupOrder", seqID, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error()))
	}
	claims, appErr := p.authorize(authToken, domain.RoleDrone)
	if appErr != nil {
		return p.writeException(ctx, out, "PickupOrder", seqID, appErr)
	}
	order, err := p.svc.DronePickup(ctx, claims.Subject, orderID)
	if err != nil {
		return p.writeException(ctx, out, "PickupOrder", seqID, mapError(err))
	}
	return p.writeReply(ctx, out, "PickupOrder", seqID, func(out thrift.TProtocol) error {
		if err := out.WriteFieldBegin(ctx, "success", thrift.STRUCT, 0); err != nil {
			return err
		}
		return writeOrder(ctx, out, order)
	})
}

func (p *Processor) handleDeliverOrder(ctx context.Context, seqID int32, in, out thrift.TProtocol) (bool, thrift.TException) {
	authToken, orderID, err := readOrderIDRequest(ctx, in)
	if err != nil {
		return p.writeException(ctx, out, "DeliverOrder", seqID, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error()))
	}
	claims, appErr := p.authorize(authToken, domain.RoleDrone)
	if appErr != nil {
		return p.writeException(ctx, out, "DeliverOrder", seqID, appErr)
	}
	order, err := p.svc.DroneDeliver(ctx, claims.Subject, orderID)
	if err != nil {
		return p.writeException(ctx, out, "DeliverOrder", seqID, mapError(err))
	}
	return p.writeReply(ctx, out, "DeliverOrder", seqID, func(out thrift.TProtocol) error {
		if err := out.WriteFieldBegin(ctx, "success", thrift.STRUCT, 0); err != nil {
			return err
		}
		return writeOrder(ctx, out, order)
	})
}

func (p *Processor) handleFailOrder(ctx context.Context, seqID int32, in, out thrift.TProtocol) (bool, thrift.TException) {
	authToken, orderID, reason, err := readFailOrderRequest(ctx, in)
	if err != nil {
		return p.writeException(ctx, out, "FailOrder", seqID, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error()))
	}
	claims, appErr := p.authorize(authToken, domain.RoleDrone)
	if appErr != nil {
		return p.writeException(ctx, out, "FailOrder", seqID, appErr)
	}
	order, err := p.svc.DroneFail(ctx, claims.Subject, orderID, reason)
	if err != nil {
		return p.writeException(ctx, out, "FailOrder", seqID, mapError(err))
	}
	return p.writeReply(ctx, out, "FailOrder", seqID, func(out thrift.TProtocol) error {
		if err := out.WriteFieldBegin(ctx, "success", thrift.STRUCT, 0); err != nil {
			return err
		}
		return writeOrder(ctx, out, order)
	})
}

func (p *Processor) handleMarkBroken(ctx context.Context, seqID int32, in, out thrift.TProtocol) (bool, thrift.TException) {
	authToken, err := readAuthRequest(ctx, in)
	if err != nil {
		return p.writeException(ctx, out, "MarkBroken", seqID, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error()))
	}
	claims, appErr := p.authorize(authToken, domain.RoleDrone)
	if appErr != nil {
		return p.writeException(ctx, out, "MarkBroken", seqID, appErr)
	}
	drone, err := p.svc.DroneMarkBroken(ctx, claims.Subject)
	if err != nil {
		return p.writeException(ctx, out, "MarkBroken", seqID, mapError(err))
	}
	return p.writeReply(ctx, out, "MarkBroken", seqID, func(out thrift.TProtocol) error {
		if err := out.WriteFieldBegin(ctx, "success", thrift.STRUCT, 0); err != nil {
			return err
		}
		return writeDrone(ctx, out, drone)
	})
}

func (p *Processor) handleHeartbeat(ctx context.Context, seqID int32, in, out thrift.TProtocol) (bool, thrift.TException) {
	authToken, loc, err := readHeartbeatRequest(ctx, in)
	if err != nil {
		return p.writeException(ctx, out, "Heartbeat", seqID, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error()))
	}
	claims, appErr := p.authorize(authToken, domain.RoleDrone)
	if appErr != nil {
		return p.writeException(ctx, out, "Heartbeat", seqID, appErr)
	}
	view, err := p.svc.DroneHeartbeat(ctx, claims.Subject, loc)
	if err != nil {
		return p.writeException(ctx, out, "Heartbeat", seqID, mapError(err))
	}
	return p.writeReply(ctx, out, "Heartbeat", seqID, func(out thrift.TProtocol) error {
		if err := out.WriteFieldBegin(ctx, "success", thrift.STRUCT, 0); err != nil {
			return err
		}
		return writeDroneStatus(ctx, out, view)
	})
}

func (p *Processor) handleCurrentOrder(ctx context.Context, seqID int32, in, out thrift.TProtocol) (bool, thrift.TException) {
	authToken, err := readAuthRequest(ctx, in)
	if err != nil {
		return p.writeException(ctx, out, "CurrentOrder", seqID, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error()))
	}
	claims, appErr := p.authorize(authToken, domain.RoleDrone)
	if appErr != nil {
		return p.writeException(ctx, out, "CurrentOrder", seqID, appErr)
	}
	view, err := p.svc.DroneCurrentOrder(ctx, claims.Subject)
	if err != nil {
		return p.writeException(ctx, out, "CurrentOrder", seqID, mapError(err))
	}
	return p.writeReply(ctx, out, "CurrentOrder", seqID, func(out thrift.TProtocol) error {
		if err := out.WriteFieldBegin(ctx, "success", thrift.STRUCT, 0); err != nil {
			return err
		}
		return writeOrderView(ctx, out, view)
	})
}

func (p *Processor) handleAdminListOrders(ctx context.Context, seqID int32, in, out thrift.TProtocol) (bool, thrift.TException) {
	authToken, status, limit, offset, err := readListOrdersRequest(ctx, in)
	if err != nil {
		return p.writeException(ctx, out, "ListOrders", seqID, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error()))
	}
	if _, appErr := p.authorize(authToken, domain.RoleAdmin); appErr != nil {
		return p.writeException(ctx, out, "ListOrders", seqID, appErr)
	}
	var st *domain.OrderStatus
	if status != "" {
		statusVal := domain.OrderStatus(status)
		st = &statusVal
	}
	views, err := p.svc.AdminListOrders(ctx, service.OrderFilter{Status: st, Limit: limit, Offset: offset})
	if err != nil {
		return p.writeException(ctx, out, "ListOrders", seqID, mapError(err))
	}
	return p.writeReply(ctx, out, "ListOrders", seqID, func(out thrift.TProtocol) error {
		if err := out.WriteFieldBegin(ctx, "success", thrift.LIST, 0); err != nil {
			return err
		}
		return writeOrderViewList(ctx, out, views)
	})
}

func (p *Processor) handleAdminUpdateOrder(ctx context.Context, seqID int32, in, out thrift.TProtocol) (bool, thrift.TException) {
	authToken, orderID, origin, dest, err := readUpdateOrderRequest(ctx, in)
	if err != nil {
		return p.writeException(ctx, out, "UpdateOrder", seqID, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error()))
	}
	if _, appErr := p.authorize(authToken, domain.RoleAdmin); appErr != nil {
		return p.writeException(ctx, out, "UpdateOrder", seqID, appErr)
	}
	order, err := p.svc.AdminUpdateOrder(ctx, orderID, origin, dest)
	if err != nil {
		return p.writeException(ctx, out, "UpdateOrder", seqID, mapError(err))
	}
	return p.writeReply(ctx, out, "UpdateOrder", seqID, func(out thrift.TProtocol) error {
		if err := out.WriteFieldBegin(ctx, "success", thrift.STRUCT, 0); err != nil {
			return err
		}
		return writeOrder(ctx, out, order)
	})
}

func (p *Processor) handleAdminListDrones(ctx context.Context, seqID int32, in, out thrift.TProtocol) (bool, thrift.TException) {
	authToken, err := readAuthRequest(ctx, in)
	if err != nil {
		return p.writeException(ctx, out, "ListDrones", seqID, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error()))
	}
	if _, appErr := p.authorize(authToken, domain.RoleAdmin); appErr != nil {
		return p.writeException(ctx, out, "ListDrones", seqID, appErr)
	}
	drones, err := p.svc.AdminListDrones(ctx)
	if err != nil {
		return p.writeException(ctx, out, "ListDrones", seqID, mapError(err))
	}
	return p.writeReply(ctx, out, "ListDrones", seqID, func(out thrift.TProtocol) error {
		if err := out.WriteFieldBegin(ctx, "success", thrift.LIST, 0); err != nil {
			return err
		}
		return writeDroneList(ctx, out, drones)
	})
}

func (p *Processor) handleAdminMarkDroneBroken(ctx context.Context, seqID int32, in, out thrift.TProtocol) (bool, thrift.TException) {
	authToken, droneID, err := readDroneIDRequest(ctx, in)
	if err != nil {
		return p.writeException(ctx, out, "MarkDroneBroken", seqID, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error()))
	}
	if _, appErr := p.authorize(authToken, domain.RoleAdmin); appErr != nil {
		return p.writeException(ctx, out, "MarkDroneBroken", seqID, appErr)
	}
	drone, err := p.svc.AdminMarkDroneBroken(ctx, droneID)
	if err != nil {
		return p.writeException(ctx, out, "MarkDroneBroken", seqID, mapError(err))
	}
	return p.writeReply(ctx, out, "MarkDroneBroken", seqID, func(out thrift.TProtocol) error {
		if err := out.WriteFieldBegin(ctx, "success", thrift.STRUCT, 0); err != nil {
			return err
		}
		return writeDrone(ctx, out, drone)
	})
}

func (p *Processor) handleAdminMarkDroneFixed(ctx context.Context, seqID int32, in, out thrift.TProtocol) (bool, thrift.TException) {
	authToken, droneID, err := readDroneIDRequest(ctx, in)
	if err != nil {
		return p.writeException(ctx, out, "MarkDroneFixed", seqID, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error()))
	}
	if _, appErr := p.authorize(authToken, domain.RoleAdmin); appErr != nil {
		return p.writeException(ctx, out, "MarkDroneFixed", seqID, appErr)
	}
	drone, err := p.svc.AdminMarkDroneFixed(ctx, droneID)
	if err != nil {
		return p.writeException(ctx, out, "MarkDroneFixed", seqID, mapError(err))
	}
	return p.writeReply(ctx, out, "MarkDroneFixed", seqID, func(out thrift.TProtocol) error {
		if err := out.WriteFieldBegin(ctx, "success", thrift.STRUCT, 0); err != nil {
			return err
		}
		return writeDrone(ctx, out, drone)
	})
}

func (p *Processor) authorize(token, role string) (*auth.Claims, thrift.TApplicationException) {
	claims, err := p.auth.ParseToken(token)
	if err != nil {
		return nil, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, "unauthorized")
	}
	if claims.Role != role {
		return nil, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, "forbidden")
	}
	return claims, nil
}

func (p *Processor) authorizeAny(token string) (*auth.Claims, thrift.TApplicationException) {
	claims, err := p.auth.ParseToken(token)
	if err != nil {
		return nil, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, "unauthorized")
	}
	return claims, nil
}

func (p *Processor) writeReply(ctx context.Context, out thrift.TProtocol, method string, seqID int32, writeSuccess func(out thrift.TProtocol) error) (bool, thrift.TException) {
	if err := out.WriteMessageBegin(ctx, method, thrift.REPLY, seqID); err != nil {
		return false, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error())
	}
	if err := out.WriteStructBegin(ctx, method+"_result"); err != nil {
		return false, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error())
	}
	if err := writeSuccess(out); err != nil {
		return false, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error())
	}
	if err := out.WriteFieldEnd(ctx); err != nil {
		return false, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error())
	}
	if err := out.WriteFieldStop(ctx); err != nil {
		return false, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error())
	}
	if err := out.WriteStructEnd(ctx); err != nil {
		return false, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error())
	}
	if err := out.WriteMessageEnd(ctx); err != nil {
		return false, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error())
	}
	if err := out.Flush(ctx); err != nil {
		return false, thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error())
	}
	return true, nil
}

func (p *Processor) writeException(ctx context.Context, out thrift.TProtocol, method string, seqID int32, appErr thrift.TApplicationException) (bool, thrift.TException) {
	_ = out.WriteMessageBegin(ctx, method, thrift.EXCEPTION, seqID)
	_ = appErr.Write(ctx, out)
	_ = out.WriteMessageEnd(ctx)
	_ = out.Flush(ctx)
	return false, appErr
}

func mapError(err error) thrift.TApplicationException {
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		return thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, "unauthorized")
	case errors.Is(err, domain.ErrForbidden):
		return thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, "forbidden")
	case errors.Is(err, domain.ErrNotFound):
		return thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, "not found")
	case errors.Is(err, domain.ErrConflict):
		return thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, "conflict")
	case errors.Is(err, domain.ErrInvalid):
		return thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, "invalid request")
	case errors.Is(err, domain.ErrPrecondition):
		return thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, "precondition failed")
	case errors.Is(err, domain.ErrNoJob):
		return thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, "no job")
	default:
		return thrift.NewTApplicationException(thrift.INTERNAL_ERROR, "internal error")
	}
}

func writeTokenResponse(ctx context.Context, out thrift.TProtocol, token string, exp time.Time) error {
	if err := out.WriteStructBegin(ctx, "TokenResponse"); err != nil {
		return err
	}
	if err := out.WriteFieldBegin(ctx, "token", thrift.STRING, 1); err != nil {
		return err
	}
	if err := out.WriteString(ctx, token); err != nil {
		return err
	}
	if err := out.WriteFieldEnd(ctx); err != nil {
		return err
	}
	if err := out.WriteFieldBegin(ctx, "expiresAt", thrift.I64, 2); err != nil {
		return err
	}
	if err := out.WriteI64(ctx, exp.Unix()); err != nil {
		return err
	}
	if err := out.WriteFieldEnd(ctx); err != nil {
		return err
	}
	if err := out.WriteFieldStop(ctx); err != nil {
		return err
	}
	return out.WriteStructEnd(ctx)
}

func writeOrder(ctx context.Context, out thrift.TProtocol, order *domain.Order) error {
	if err := out.WriteStructBegin(ctx, "Order"); err != nil {
		return err
	}
	if err := out.WriteFieldBegin(ctx, "id", thrift.STRING, 1); err != nil {
		return err
	}
	if err := out.WriteString(ctx, order.ID); err != nil {
		return err
	}
	if err := out.WriteFieldEnd(ctx); err != nil {
		return err
	}
	if err := out.WriteFieldBegin(ctx, "userId", thrift.STRING, 2); err != nil {
		return err
	}
	if err := out.WriteString(ctx, order.UserID); err != nil {
		return err
	}
	if err := out.WriteFieldEnd(ctx); err != nil {
		return err
	}
	if err := out.WriteFieldBegin(ctx, "origin", thrift.STRUCT, 3); err != nil {
		return err
	}
	if err := writeLocation(ctx, out, order.Origin); err != nil {
		return err
	}
	if err := out.WriteFieldEnd(ctx); err != nil {
		return err
	}
	if err := out.WriteFieldBegin(ctx, "destination", thrift.STRUCT, 4); err != nil {
		return err
	}
	if err := writeLocation(ctx, out, order.Destination); err != nil {
		return err
	}
	if err := out.WriteFieldEnd(ctx); err != nil {
		return err
	}
	if err := out.WriteFieldBegin(ctx, "status", thrift.STRING, 5); err != nil {
		return err
	}
	if err := out.WriteString(ctx, string(order.Status)); err != nil {
		return err
	}
	if err := out.WriteFieldEnd(ctx); err != nil {
		return err
	}
	if order.AssignedDroneID != nil {
		if err := out.WriteFieldBegin(ctx, "assignedDroneId", thrift.STRING, 6); err != nil {
			return err
		}
		if err := out.WriteString(ctx, *order.AssignedDroneID); err != nil {
			return err
		}
		if err := out.WriteFieldEnd(ctx); err != nil {
			return err
		}
	}
	if order.HandoffOrigin != nil {
		if err := out.WriteFieldBegin(ctx, "handoffOrigin", thrift.STRUCT, 7); err != nil {
			return err
		}
		if err := writeLocation(ctx, out, *order.HandoffOrigin); err != nil {
			return err
		}
		if err := out.WriteFieldEnd(ctx); err != nil {
			return err
		}
	}
	if err := out.WriteFieldBegin(ctx, "createdAt", thrift.I64, 8); err != nil {
		return err
	}
	if err := out.WriteI64(ctx, order.CreatedAt.Unix()); err != nil {
		return err
	}
	if err := out.WriteFieldEnd(ctx); err != nil {
		return err
	}
	if err := out.WriteFieldBegin(ctx, "updatedAt", thrift.I64, 9); err != nil {
		return err
	}
	if err := out.WriteI64(ctx, order.UpdatedAt.Unix()); err != nil {
		return err
	}
	if err := out.WriteFieldEnd(ctx); err != nil {
		return err
	}
	if order.ReservedAt != nil {
		if err := out.WriteFieldBegin(ctx, "reservedAt", thrift.I64, 10); err != nil {
			return err
		}
		if err := out.WriteI64(ctx, order.ReservedAt.Unix()); err != nil {
			return err
		}
		if err := out.WriteFieldEnd(ctx); err != nil {
			return err
		}
	}
	if order.PickedUpAt != nil {
		if err := out.WriteFieldBegin(ctx, "pickedUpAt", thrift.I64, 11); err != nil {
			return err
		}
		if err := out.WriteI64(ctx, order.PickedUpAt.Unix()); err != nil {
			return err
		}
		if err := out.WriteFieldEnd(ctx); err != nil {
			return err
		}
	}
	if order.DeliveredAt != nil {
		if err := out.WriteFieldBegin(ctx, "deliveredAt", thrift.I64, 12); err != nil {
			return err
		}
		if err := out.WriteI64(ctx, order.DeliveredAt.Unix()); err != nil {
			return err
		}
		if err := out.WriteFieldEnd(ctx); err != nil {
			return err
		}
	}
	if order.FailedAt != nil {
		if err := out.WriteFieldBegin(ctx, "failedAt", thrift.I64, 13); err != nil {
			return err
		}
		if err := out.WriteI64(ctx, order.FailedAt.Unix()); err != nil {
			return err
		}
		if err := out.WriteFieldEnd(ctx); err != nil {
			return err
		}
	}
	if order.FailureReason != nil {
		if err := out.WriteFieldBegin(ctx, "failureReason", thrift.STRING, 14); err != nil {
			return err
		}
		if err := out.WriteString(ctx, *order.FailureReason); err != nil {
			return err
		}
		if err := out.WriteFieldEnd(ctx); err != nil {
			return err
		}
	}
	return out.WriteStructEnd(ctx)
}

func writeOrderView(ctx context.Context, out thrift.TProtocol, view *service.OrderView) error {
	if err := out.WriteStructBegin(ctx, "OrderView"); err != nil {
		return err
	}
	if err := out.WriteFieldBegin(ctx, "order", thrift.STRUCT, 1); err != nil {
		return err
	}
	if err := writeOrder(ctx, out, view.Order); err != nil {
		return err
	}
	if err := out.WriteFieldEnd(ctx); err != nil {
		return err
	}
	if view.CurrentLocation != nil {
		if err := out.WriteFieldBegin(ctx, "currentLocation", thrift.STRUCT, 2); err != nil {
			return err
		}
		if err := writeLocation(ctx, out, *view.CurrentLocation); err != nil {
			return err
		}
		if err := out.WriteFieldEnd(ctx); err != nil {
			return err
		}
	}
	if view.ETASeconds != nil {
		if err := out.WriteFieldBegin(ctx, "etaSeconds", thrift.I64, 3); err != nil {
			return err
		}
		if err := out.WriteI64(ctx, *view.ETASeconds); err != nil {
			return err
		}
		if err := out.WriteFieldEnd(ctx); err != nil {
			return err
		}
	}
	return out.WriteStructEnd(ctx)
}

func writeDrone(ctx context.Context, out thrift.TProtocol, drone *domain.Drone) error {
	if err := out.WriteStructBegin(ctx, "Drone"); err != nil {
		return err
	}
	if err := out.WriteFieldBegin(ctx, "id", thrift.STRING, 1); err != nil {
		return err
	}
	if err := out.WriteString(ctx, drone.ID); err != nil {
		return err
	}
	if err := out.WriteFieldEnd(ctx); err != nil {
		return err
	}
	if err := out.WriteFieldBegin(ctx, "status", thrift.STRING, 2); err != nil {
		return err
	}
	if err := out.WriteString(ctx, string(drone.Status)); err != nil {
		return err
	}
	if err := out.WriteFieldEnd(ctx); err != nil {
		return err
	}
	if drone.LastLocation != nil {
		if err := out.WriteFieldBegin(ctx, "lastLocation", thrift.STRUCT, 3); err != nil {
			return err
		}
		if err := writeLocation(ctx, out, *drone.LastLocation); err != nil {
			return err
		}
		if err := out.WriteFieldEnd(ctx); err != nil {
			return err
		}
	}
	if drone.LastHeartbeatAt != nil {
		if err := out.WriteFieldBegin(ctx, "lastHeartbeatAt", thrift.I64, 4); err != nil {
			return err
		}
		if err := out.WriteI64(ctx, drone.LastHeartbeatAt.Unix()); err != nil {
			return err
		}
		if err := out.WriteFieldEnd(ctx); err != nil {
			return err
		}
	}
	if drone.CurrentOrderID != nil {
		if err := out.WriteFieldBegin(ctx, "currentOrderId", thrift.STRING, 5); err != nil {
			return err
		}
		if err := out.WriteString(ctx, *drone.CurrentOrderID); err != nil {
			return err
		}
		if err := out.WriteFieldEnd(ctx); err != nil {
			return err
		}
	}
	if err := out.WriteFieldBegin(ctx, "createdAt", thrift.I64, 6); err != nil {
		return err
	}
	if err := out.WriteI64(ctx, drone.CreatedAt.Unix()); err != nil {
		return err
	}
	if err := out.WriteFieldEnd(ctx); err != nil {
		return err
	}
	if err := out.WriteFieldBegin(ctx, "updatedAt", thrift.I64, 7); err != nil {
		return err
	}
	if err := out.WriteI64(ctx, drone.UpdatedAt.Unix()); err != nil {
		return err
	}
	if err := out.WriteFieldEnd(ctx); err != nil {
		return err
	}
	return out.WriteStructEnd(ctx)
}

func writeDroneStatus(ctx context.Context, out thrift.TProtocol, view *service.DroneStatusView) error {
	if err := out.WriteStructBegin(ctx, "DroneStatus"); err != nil {
		return err
	}
	if err := out.WriteFieldBegin(ctx, "drone", thrift.STRUCT, 1); err != nil {
		return err
	}
	if err := writeDrone(ctx, out, view.Drone); err != nil {
		return err
	}
	if err := out.WriteFieldEnd(ctx); err != nil {
		return err
	}
	if view.CurrentOrder != nil {
		if err := out.WriteFieldBegin(ctx, "currentOrder", thrift.STRUCT, 2); err != nil {
			return err
		}
		if err := writeOrderView(ctx, out, view.CurrentOrder); err != nil {
			return err
		}
		if err := out.WriteFieldEnd(ctx); err != nil {
			return err
		}
	}
	return out.WriteStructEnd(ctx)
}

func writeOrderViewList(ctx context.Context, out thrift.TProtocol, views []*service.OrderView) error {
	if err := out.WriteListBegin(ctx, thrift.STRUCT, len(views)); err != nil {
		return err
	}
	for _, view := range views {
		if err := writeOrderView(ctx, out, view); err != nil {
			return err
		}
	}
	return out.WriteListEnd(ctx)
}

func writeDroneList(ctx context.Context, out thrift.TProtocol, drones []*domain.Drone) error {
	if err := out.WriteListBegin(ctx, thrift.STRUCT, len(drones)); err != nil {
		return err
	}
	for _, drone := range drones {
		if err := writeDrone(ctx, out, drone); err != nil {
			return err
		}
	}
	return out.WriteListEnd(ctx)
}

func writeLocation(ctx context.Context, out thrift.TProtocol, loc domain.Location) error {
	if err := out.WriteStructBegin(ctx, "Location"); err != nil {
		return err
	}
	if err := out.WriteFieldBegin(ctx, "lat", thrift.DOUBLE, 1); err != nil {
		return err
	}
	if err := out.WriteDouble(ctx, loc.Lat); err != nil {
		return err
	}
	if err := out.WriteFieldEnd(ctx); err != nil {
		return err
	}
	if err := out.WriteFieldBegin(ctx, "lng", thrift.DOUBLE, 2); err != nil {
		return err
	}
	if err := out.WriteDouble(ctx, loc.Lng); err != nil {
		return err
	}
	if err := out.WriteFieldEnd(ctx); err != nil {
		return err
	}
	return out.WriteStructEnd(ctx)
}

func readTokenRequest(ctx context.Context, in thrift.TProtocol) (string, string, error) {
	// Expected args struct: IssueToken_args { 1: TokenRequest request }
	if _, err := in.ReadStructBegin(ctx); err != nil {
		return "", "", err
	}
	var name, role string
	for {
		_, fieldType, fieldID, err := in.ReadFieldBegin(ctx)
		if err != nil {
			return "", "", err
		}
		if fieldType == thrift.STOP {
			break
		}
		if fieldID == 1 && fieldType == thrift.STRUCT {
			// TokenRequest { 1:name, 2:role }
			if _, err := in.ReadStructBegin(ctx); err != nil {
				return "", "", err
			}
			for {
				_, ft, fid, err := in.ReadFieldBegin(ctx)
				if err != nil {
					return "", "", err
				}
				if ft == thrift.STOP {
					break
				}
				switch fid {
				case 1:
					name, err = in.ReadString(ctx)
				case 2:
					role, err = in.ReadString(ctx)
				default:
					err = in.Skip(ctx, ft)
				}
				if err != nil {
					return "", "", err
				}
				if err := in.ReadFieldEnd(ctx); err != nil {
					return "", "", err
				}
			}
			if err := in.ReadStructEnd(ctx); err != nil {
				return "", "", err
			}
		} else {
			if err := in.Skip(ctx, fieldType); err != nil {
				return "", "", err
			}
		}
		if err := in.ReadFieldEnd(ctx); err != nil {
			return "", "", err
		}
	}
	if err := in.ReadStructEnd(ctx); err != nil {
		return "", "", err
	}
	if err := in.ReadMessageEnd(ctx); err != nil {
		return "", "", err
	}
	return name, role, nil
}

func readAuthRequest(ctx context.Context, in thrift.TProtocol) (string, error) {
	// Expected args struct: <Method>_args { 1: AuthRequest request }
	if _, err := in.ReadStructBegin(ctx); err != nil {
		return "", err
	}
	var token string
	for {
		_, fieldType, fieldID, err := in.ReadFieldBegin(ctx)
		if err != nil {
			return "", err
		}
		if fieldType == thrift.STOP {
			break
		}
		if fieldID == 1 && fieldType == thrift.STRUCT {
			if _, err := in.ReadStructBegin(ctx); err != nil {
				return "", err
			}
			for {
				_, ft, fid, err := in.ReadFieldBegin(ctx)
				if err != nil {
					return "", err
				}
				if ft == thrift.STOP {
					break
				}
				if fid == 1 {
					token, err = in.ReadString(ctx)
				} else {
					err = in.Skip(ctx, ft)
				}
				if err != nil {
					return "", err
				}
				if err := in.ReadFieldEnd(ctx); err != nil {
					return "", err
				}
			}
			if err := in.ReadStructEnd(ctx); err != nil {
				return "", err
			}
		} else {
			if err := in.Skip(ctx, fieldType); err != nil {
				return "", err
			}
		}
		if err := in.ReadFieldEnd(ctx); err != nil {
			return "", err
		}
	}
	if err := in.ReadStructEnd(ctx); err != nil {
		return "", err
	}
	if err := in.ReadMessageEnd(ctx); err != nil {
		return "", err
	}
	return token, nil
}

func readOrderIDRequest(ctx context.Context, in thrift.TProtocol) (string, string, error) {
	// Expected args struct: <Method>_args { 1: OrderIDRequest request }
	if _, err := in.ReadStructBegin(ctx); err != nil {
		return "", "", err
	}
	var token, orderID string
	for {
		_, fieldType, fieldID, err := in.ReadFieldBegin(ctx)
		if err != nil {
			return "", "", err
		}
		if fieldType == thrift.STOP {
			break
		}
		if fieldID == 1 && fieldType == thrift.STRUCT {
			if _, err := in.ReadStructBegin(ctx); err != nil {
				return "", "", err
			}
			for {
				_, ft, fid, err := in.ReadFieldBegin(ctx)
				if err != nil {
					return "", "", err
				}
				if ft == thrift.STOP {
					break
				}
				switch fid {
				case 1:
					token, err = in.ReadString(ctx)
				case 2:
					orderID, err = in.ReadString(ctx)
				default:
					err = in.Skip(ctx, ft)
				}
				if err != nil {
					return "", "", err
				}
				if err := in.ReadFieldEnd(ctx); err != nil {
					return "", "", err
				}
			}
			if err := in.ReadStructEnd(ctx); err != nil {
				return "", "", err
			}
		} else {
			if err := in.Skip(ctx, fieldType); err != nil {
				return "", "", err
			}
		}
		if err := in.ReadFieldEnd(ctx); err != nil {
			return "", "", err
		}
	}
	if err := in.ReadStructEnd(ctx); err != nil {
		return "", "", err
	}
	if err := in.ReadMessageEnd(ctx); err != nil {
		return "", "", err
	}
	return token, orderID, nil
}

func readDroneIDRequest(ctx context.Context, in thrift.TProtocol) (string, string, error) {
	return readOrderIDRequest(ctx, in)
}

func readFailOrderRequest(ctx context.Context, in thrift.TProtocol) (string, string, string, error) {
	// Expected args struct: FailOrder_args { 1: FailOrderRequest request }
	if _, err := in.ReadStructBegin(ctx); err != nil {
		return "", "", "", err
	}
	var token, orderID, reason string
	for {
		_, fieldType, fieldID, err := in.ReadFieldBegin(ctx)
		if err != nil {
			return "", "", "", err
		}
		if fieldType == thrift.STOP {
			break
		}
		if fieldID == 1 && fieldType == thrift.STRUCT {
			if _, err := in.ReadStructBegin(ctx); err != nil {
				return "", "", "", err
			}
			for {
				_, ft, fid, err := in.ReadFieldBegin(ctx)
				if err != nil {
					return "", "", "", err
				}
				if ft == thrift.STOP {
					break
				}
				switch fid {
				case 1:
					token, err = in.ReadString(ctx)
				case 2:
					orderID, err = in.ReadString(ctx)
				case 3:
					reason, err = in.ReadString(ctx)
				default:
					err = in.Skip(ctx, ft)
				}
				if err != nil {
					return "", "", "", err
				}
				if err := in.ReadFieldEnd(ctx); err != nil {
					return "", "", "", err
				}
			}
			if err := in.ReadStructEnd(ctx); err != nil {
				return "", "", "", err
			}
		} else {
			if err := in.Skip(ctx, fieldType); err != nil {
				return "", "", "", err
			}
		}
		if err := in.ReadFieldEnd(ctx); err != nil {
			return "", "", "", err
		}
	}
	if err := in.ReadStructEnd(ctx); err != nil {
		return "", "", "", err
	}
	if err := in.ReadMessageEnd(ctx); err != nil {
		return "", "", "", err
	}
	return token, orderID, reason, nil
}

func readHeartbeatRequest(ctx context.Context, in thrift.TProtocol) (string, domain.Location, error) {
	// Expected args struct: Heartbeat_args { 1: HeartbeatRequest request }
	if _, err := in.ReadStructBegin(ctx); err != nil {
		return "", domain.Location{}, err
	}
	var token string
	var loc domain.Location
	for {
		_, fieldType, fieldID, err := in.ReadFieldBegin(ctx)
		if err != nil {
			return "", domain.Location{}, err
		}
		if fieldType == thrift.STOP {
			break
		}
		if fieldID == 1 && fieldType == thrift.STRUCT {
			if _, err := in.ReadStructBegin(ctx); err != nil {
				return "", domain.Location{}, err
			}
			for {
				_, ft, fid, err := in.ReadFieldBegin(ctx)
				if err != nil {
					return "", domain.Location{}, err
				}
				if ft == thrift.STOP {
					break
				}
				switch fid {
				case 1:
					token, err = in.ReadString(ctx)
				case 2:
					// location struct
					if ft != thrift.STRUCT {
						err = in.Skip(ctx, ft)
					} else {
						var l domain.Location
						l, err = readLocation(ctx, in)
						loc = l
					}
				default:
					err = in.Skip(ctx, ft)
				}
				if err != nil {
					return "", domain.Location{}, err
				}
				if err := in.ReadFieldEnd(ctx); err != nil {
					return "", domain.Location{}, err
				}
			}
			if err := in.ReadStructEnd(ctx); err != nil {
				return "", domain.Location{}, err
			}
		} else {
			if err := in.Skip(ctx, fieldType); err != nil {
				return "", domain.Location{}, err
			}
		}
		if err := in.ReadFieldEnd(ctx); err != nil {
			return "", domain.Location{}, err
		}
	}
	if err := in.ReadStructEnd(ctx); err != nil {
		return "", domain.Location{}, err
	}
	if err := in.ReadMessageEnd(ctx); err != nil {
		return "", domain.Location{}, err
	}
	return token, loc, nil
}

func readSubmitOrderRequest(ctx context.Context, in thrift.TProtocol) (string, domain.Location, domain.Location, error) {
	if _, err := in.ReadStructBegin(ctx); err != nil {
		return "", domain.Location{}, domain.Location{}, err
	}
	var token string
	var origin, dest domain.Location
	for {
		_, fieldType, fieldID, err := in.ReadFieldBegin(ctx)
		if err != nil {
			return "", domain.Location{}, domain.Location{}, err
		}
		if fieldType == thrift.STOP {
			break
		}
		switch fieldID {
		case 1:
			token, err = in.ReadString(ctx)
		case 2:
			origin, err = readLocation(ctx, in)
		case 3:
			dest, err = readLocation(ctx, in)
		default:
			err = in.Skip(ctx, fieldType)
		}
		if err != nil {
			return "", domain.Location{}, domain.Location{}, err
		}
		if err := in.ReadFieldEnd(ctx); err != nil {
			return "", domain.Location{}, domain.Location{}, err
		}
	}
	if err := in.ReadStructEnd(ctx); err != nil {
		return "", domain.Location{}, domain.Location{}, err
	}
	if err := in.ReadMessageEnd(ctx); err != nil {
		return "", domain.Location{}, domain.Location{}, err
	}
	return token, origin, dest, nil
}

func readListOrdersRequest(ctx context.Context, in thrift.TProtocol) (string, string, int, int, error) {
	if _, err := in.ReadStructBegin(ctx); err != nil {
		return "", "", 0, 0, err
	}
	var token string
	var status string
	var limit int32
	var offset int32
	for {
		_, fieldType, fieldID, err := in.ReadFieldBegin(ctx)
		if err != nil {
			return "", "", 0, 0, err
		}
		if fieldType == thrift.STOP {
			break
		}
		switch fieldID {
		case 1:
			token, err = in.ReadString(ctx)
		case 2:
			status, err = in.ReadString(ctx)
		case 3:
			limit, err = in.ReadI32(ctx)
		case 4:
			offset, err = in.ReadI32(ctx)
		default:
			err = in.Skip(ctx, fieldType)
		}
		if err != nil {
			return "", "", 0, 0, err
		}
		if err := in.ReadFieldEnd(ctx); err != nil {
			return "", "", 0, 0, err
		}
	}
	if err := in.ReadStructEnd(ctx); err != nil {
		return "", "", 0, 0, err
	}
	if err := in.ReadMessageEnd(ctx); err != nil {
		return "", "", 0, 0, err
	}
	return token, status, int(limit), int(offset), nil
}

func readUpdateOrderRequest(ctx context.Context, in thrift.TProtocol) (string, string, *domain.Location, *domain.Location, error) {
	if _, err := in.ReadStructBegin(ctx); err != nil {
		return "", "", nil, nil, err
	}
	var token, orderID string
	var origin *domain.Location
	var dest *domain.Location
	for {
		_, fieldType, fieldID, err := in.ReadFieldBegin(ctx)
		if err != nil {
			return "", "", nil, nil, err
		}
		if fieldType == thrift.STOP {
			break
		}
		switch fieldID {
		case 1:
			token, err = in.ReadString(ctx)
		case 2:
			orderID, err = in.ReadString(ctx)
		case 3:
			loc, err := readLocation(ctx, in)
			if err != nil {
				return "", "", nil, nil, err
			}
			origin = &loc
		case 4:
			loc, err := readLocation(ctx, in)
			if err != nil {
				return "", "", nil, nil, err
			}
			dest = &loc
		default:
			err = in.Skip(ctx, fieldType)
		}
		if err != nil {
			return "", "", nil, nil, err
		}
		if err := in.ReadFieldEnd(ctx); err != nil {
			return "", "", nil, nil, err
		}
	}
	if err := in.ReadStructEnd(ctx); err != nil {
		return "", "", nil, nil, err
	}
	if err := in.ReadMessageEnd(ctx); err != nil {
		return "", "", nil, nil, err
	}
	return token, orderID, origin, dest, nil
}

func readLocation(ctx context.Context, in thrift.TProtocol) (domain.Location, error) {
	if _, err := in.ReadStructBegin(ctx); err != nil {
		return domain.Location{}, err
	}
	var lat, lng float64
	for {
		_, fieldType, fieldID, err := in.ReadFieldBegin(ctx)
		if err != nil {
			return domain.Location{}, err
		}
		if fieldType == thrift.STOP {
			break
		}
		switch fieldID {
		case 1:
			lat, err = in.ReadDouble(ctx)
		case 2:
			lng, err = in.ReadDouble(ctx)
		default:
			err = in.Skip(ctx, fieldType)
		}
		if err != nil {
			return domain.Location{}, err
		}
		if err := in.ReadFieldEnd(ctx); err != nil {
			return domain.Location{}, err
		}
	}
	if err := in.ReadStructEnd(ctx); err != nil {
		return domain.Location{}, err
	}
	return domain.Location{Lat: lat, Lng: lng}, nil
}
