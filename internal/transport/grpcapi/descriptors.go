package grpcapi

import (
	"context"

	"google.golang.org/grpc"

	"penny-assesment/internal/transport"
)

type Empty struct{}

type ListOrdersResponse struct {
	Orders []transport.OrderViewResponse `json:"orders"`
}

type ListDronesResponse struct {
	Drones []transport.DroneResponse `json:"drones"`
}

type AuthService interface {
	IssueToken(context.Context, *TokenRequest) (*TokenResponse, error)
}

type OrderService interface {
	SubmitOrder(context.Context, *SubmitOrderRequest) (*transport.OrderResponse, error)
	WithdrawOrder(context.Context, *OrderIDRequest) (*transport.OrderResponse, error)
	GetOrder(context.Context, *OrderIDRequest) (*transport.OrderViewResponse, error)
}

type DroneService interface {
	ReserveJob(context.Context, *Empty) (*transport.OrderResponse, error)
	PickupOrder(context.Context, *OrderIDRequest) (*transport.OrderResponse, error)
	DeliverOrder(context.Context, *OrderIDRequest) (*transport.OrderResponse, error)
	FailOrder(context.Context, *FailOrderRequest) (*transport.OrderResponse, error)
	MarkDroneBroken(context.Context, *Empty) (*transport.DroneResponse, error)
	Heartbeat(context.Context, *HeartbeatRequest) (*transport.DroneStatusResponse, error)
	CurrentOrder(context.Context, *Empty) (*transport.OrderViewResponse, error)
}

type AdminService interface {
	AdminListOrders(context.Context, *ListOrdersRequest) (*ListOrdersResponse, error)
	AdminUpdateOrder(context.Context, *UpdateOrderRequest) (*transport.OrderResponse, error)
	AdminListDrones(context.Context, *Empty) (*ListDronesResponse, error)
	AdminMarkDroneBroken(context.Context, *DroneIDRequest) (*transport.DroneResponse, error)
	AdminMarkDroneFixed(context.Context, *DroneIDRequest) (*transport.DroneResponse, error)
}

var authServiceDesc = grpc.ServiceDesc{
	ServiceName: "drone.AuthService",
	HandlerType: (*AuthService)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "IssueToken",
			Handler:    issueTokenHandler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "drone_delivery.proto",
}

var droneServiceDesc = grpc.ServiceDesc{
	ServiceName: "drone.DroneService",
	HandlerType: (*DroneService)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "ReserveJob", Handler: reserveJobHandler},
		{MethodName: "PickupOrder", Handler: pickupOrderHandler},
		{MethodName: "DeliverOrder", Handler: deliverOrderHandler},
		{MethodName: "FailOrder", Handler: failOrderHandler},
		{MethodName: "MarkBroken", Handler: markBrokenHandler},
		{MethodName: "Heartbeat", Handler: heartbeatHandler},
		{MethodName: "CurrentOrder", Handler: currentOrderHandler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "drone_delivery.proto",
}

var orderServiceDesc = grpc.ServiceDesc{
	ServiceName: "drone.OrderService",
	HandlerType: (*OrderService)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "SubmitOrder", Handler: submitOrderHandler},
		{MethodName: "WithdrawOrder", Handler: withdrawOrderHandler},
		{MethodName: "GetOrder", Handler: getOrderHandler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "drone_delivery.proto",
}

var adminServiceDesc = grpc.ServiceDesc{
	ServiceName: "drone.AdminService",
	HandlerType: (*AdminService)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "ListOrders", Handler: adminListOrdersHandler},
		{MethodName: "UpdateOrder", Handler: adminUpdateOrderHandler},
		{MethodName: "ListDrones", Handler: adminListDronesHandler},
		{MethodName: "MarkDroneBroken", Handler: adminMarkDroneBrokenHandler},
		{MethodName: "MarkDroneFixed", Handler: adminMarkDroneFixedHandler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "drone_delivery.proto",
}

func issueTokenHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(TokenRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).IssueToken(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/drone.AuthService/IssueToken"}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).IssueToken(ctx, req.(*TokenRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func submitOrderHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(SubmitOrderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).SubmitOrder(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/drone.OrderService/SubmitOrder"}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).SubmitOrder(ctx, req.(*SubmitOrderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func withdrawOrderHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(OrderIDRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).WithdrawOrder(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/drone.OrderService/WithdrawOrder"}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).WithdrawOrder(ctx, req.(*OrderIDRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func getOrderHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(OrderIDRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).GetOrder(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/drone.OrderService/GetOrder"}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).GetOrder(ctx, req.(*OrderIDRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func reserveJobHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).ReserveJob(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/drone.DroneService/ReserveJob"}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).ReserveJob(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func pickupOrderHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(OrderIDRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).PickupOrder(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/drone.DroneService/PickupOrder"}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).PickupOrder(ctx, req.(*OrderIDRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func deliverOrderHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(OrderIDRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).DeliverOrder(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/drone.DroneService/DeliverOrder"}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).DeliverOrder(ctx, req.(*OrderIDRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func failOrderHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(FailOrderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).FailOrder(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/drone.DroneService/FailOrder"}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).FailOrder(ctx, req.(*FailOrderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func markBrokenHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).MarkDroneBroken(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/drone.DroneService/MarkBroken"}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).MarkDroneBroken(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func heartbeatHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(HeartbeatRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).Heartbeat(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/drone.DroneService/Heartbeat"}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).Heartbeat(ctx, req.(*HeartbeatRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func currentOrderHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).CurrentOrder(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/drone.DroneService/CurrentOrder"}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).CurrentOrder(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func adminListOrdersHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(ListOrdersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).AdminListOrders(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/drone.AdminService/ListOrders"}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).AdminListOrders(ctx, req.(*ListOrdersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func adminUpdateOrderHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(UpdateOrderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).AdminUpdateOrder(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/drone.AdminService/UpdateOrder"}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).AdminUpdateOrder(ctx, req.(*UpdateOrderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func adminListDronesHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).AdminListDrones(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/drone.AdminService/ListDrones"}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).AdminListDrones(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func adminMarkDroneBrokenHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(DroneIDRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).AdminMarkDroneBroken(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/drone.AdminService/MarkDroneBroken"}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).AdminMarkDroneBroken(ctx, req.(*DroneIDRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func adminMarkDroneFixedHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(DroneIDRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).AdminMarkDroneFixed(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/drone.AdminService/MarkDroneFixed"}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).AdminMarkDroneFixed(ctx, req.(*DroneIDRequest))
	}
	return interceptor(ctx, in, info, handler)
}
