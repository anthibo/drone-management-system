package grpcapi

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"penny-assesment/internal/auth"
	"penny-assesment/internal/domain"
	"penny-assesment/internal/service"
	"penny-assesment/internal/transport"
)

type Server struct {
	svc  *service.Service
	auth *auth.Authenticator
}

func NewServer(svc *service.Service, authenticator *auth.Authenticator) *grpc.Server {
	server := &Server{svc: svc, auth: authenticator}
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(server.authInterceptor()))

	grpcServer.RegisterService(&authServiceDesc, server)
	grpcServer.RegisterService(&droneServiceDesc, server)
	grpcServer.RegisterService(&orderServiceDesc, server)
	grpcServer.RegisterService(&adminServiceDesc, server)

	return grpcServer
}

func (s *Server) authInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if info.FullMethod == "/drone.AuthService/IssueToken" {
			return handler(ctx, req)
		}
		md, _ := metadata.FromIncomingContext(ctx)
		authHeader := ""
		if values := md.Get("authorization"); len(values) > 0 {
			authHeader = values[0]
		}
		token := auth.ExtractBearerToken(authHeader)
		if token == "" {
			return nil, status.Error(codes.Unauthenticated, "missing token")
		}
		claims, err := s.auth.ParseToken(token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}
		ctx = auth.ContextWithClaims(ctx, claims)
		return handler(ctx, req)
	}
}

func (s *Server) IssueToken(ctx context.Context, req *TokenRequest) (*TokenResponse, error) {
	if req.Name == "" || !domain.ValidateRole(req.Role) {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	token, exp, err := s.auth.IssueToken(req.Name, req.Role)
	if err != nil {
		return nil, status.Error(codes.Internal, "token error")
	}
	return &TokenResponse{Token: token, ExpiresAt: exp.Format(time.RFC3339)}, nil
}

func (s *Server) SubmitOrder(ctx context.Context, req *SubmitOrderRequest) (*transport.OrderResponse, error) {
	claims, err := requireRole(ctx, domain.RoleEndUser)
	if err != nil {
		return nil, err
	}
	order, err := s.svc.SubmitOrder(ctx, claims.Subject, toDomainLocation(req.Origin), toDomainLocation(req.Destination))
	if err != nil {
		return nil, mapServiceError(err)
	}
	resp := transport.FromOrder(order)
	return &resp, nil
}

func (s *Server) WithdrawOrder(ctx context.Context, req *OrderIDRequest) (*transport.OrderResponse, error) {
	claims, err := requireRole(ctx, domain.RoleEndUser)
	if err != nil {
		return nil, err
	}
	order, err := s.svc.WithdrawOrder(ctx, claims.Subject, req.OrderID)
	if err != nil {
		return nil, mapServiceError(err)
	}
	resp := transport.FromOrder(order)
	return &resp, nil
}

func (s *Server) GetOrder(ctx context.Context, req *OrderIDRequest) (*transport.OrderViewResponse, error) {
	claims, err := getClaims(ctx)
	if err != nil {
		return nil, err
	}
	view, err := s.svc.GetOrderView(ctx, claims.Subject, claims.Role, req.OrderID)
	if err != nil {
		return nil, mapServiceError(err)
	}
	resp := transport.FromOrderView(view)
	return &resp, nil
}

func (s *Server) ReserveJob(ctx context.Context, _ *Empty) (*transport.OrderResponse, error) {
	claims, err := requireRole(ctx, domain.RoleDrone)
	if err != nil {
		return nil, err
	}
	order, err := s.svc.DroneReserveJob(ctx, claims.Subject)
	if err != nil {
		return nil, mapServiceError(err)
	}
	resp := transport.FromOrder(order)
	return &resp, nil
}

func (s *Server) PickupOrder(ctx context.Context, req *OrderIDRequest) (*transport.OrderResponse, error) {
	claims, err := requireRole(ctx, domain.RoleDrone)
	if err != nil {
		return nil, err
	}
	order, err := s.svc.DronePickup(ctx, claims.Subject, req.OrderID)
	if err != nil {
		return nil, mapServiceError(err)
	}
	resp := transport.FromOrder(order)
	return &resp, nil
}

func (s *Server) DeliverOrder(ctx context.Context, req *OrderIDRequest) (*transport.OrderResponse, error) {
	claims, err := requireRole(ctx, domain.RoleDrone)
	if err != nil {
		return nil, err
	}
	order, err := s.svc.DroneDeliver(ctx, claims.Subject, req.OrderID)
	if err != nil {
		return nil, mapServiceError(err)
	}
	resp := transport.FromOrder(order)
	return &resp, nil
}

func (s *Server) FailOrder(ctx context.Context, req *FailOrderRequest) (*transport.OrderResponse, error) {
	claims, err := requireRole(ctx, domain.RoleDrone)
	if err != nil {
		return nil, err
	}
	order, err := s.svc.DroneFail(ctx, claims.Subject, req.OrderID, req.Reason)
	if err != nil {
		return nil, mapServiceError(err)
	}
	resp := transport.FromOrder(order)
	return &resp, nil
}

func (s *Server) MarkDroneBroken(ctx context.Context, _ *Empty) (*transport.DroneResponse, error) {
	claims, err := requireRole(ctx, domain.RoleDrone)
	if err != nil {
		return nil, err
	}
	drone, err := s.svc.DroneMarkBroken(ctx, claims.Subject)
	if err != nil {
		return nil, mapServiceError(err)
	}
	resp := transport.FromDrone(drone)
	return &resp, nil
}

func (s *Server) Heartbeat(ctx context.Context, req *HeartbeatRequest) (*transport.DroneStatusResponse, error) {
	claims, err := requireRole(ctx, domain.RoleDrone)
	if err != nil {
		return nil, err
	}
	view, err := s.svc.DroneHeartbeat(ctx, claims.Subject, domain.Location{Lat: req.Lat, Lng: req.Lng})
	if err != nil {
		return nil, mapServiceError(err)
	}
	resp := transport.FromDroneStatus(view)
	return &resp, nil
}

func (s *Server) CurrentOrder(ctx context.Context, _ *Empty) (*transport.OrderViewResponse, error) {
	claims, err := requireRole(ctx, domain.RoleDrone)
	if err != nil {
		return nil, err
	}
	view, err := s.svc.DroneCurrentOrder(ctx, claims.Subject)
	if err != nil {
		return nil, mapServiceError(err)
	}
	resp := transport.FromOrderView(view)
	return &resp, nil
}

func (s *Server) AdminListOrders(ctx context.Context, req *ListOrdersRequest) (*ListOrdersResponse, error) {
	if _, err := requireRole(ctx, domain.RoleAdmin); err != nil {
		return nil, err
	}
	var status *domain.OrderStatus
	if req.Status != "" {
		st := domain.OrderStatus(req.Status)
		status = &st
	}
	views, err := s.svc.AdminListOrders(ctx, service.OrderFilter{Status: status, Limit: req.Limit, Offset: req.Offset})
	if err != nil {
		return nil, mapServiceError(err)
	}
	resp := &ListOrdersResponse{Orders: make([]transport.OrderViewResponse, 0, len(views))}
	for _, view := range views {
		resp.Orders = append(resp.Orders, transport.FromOrderView(view))
	}
	return resp, nil
}

func (s *Server) AdminUpdateOrder(ctx context.Context, req *UpdateOrderRequest) (*transport.OrderResponse, error) {
	if _, err := requireRole(ctx, domain.RoleAdmin); err != nil {
		return nil, err
	}
	var origin *domain.Location
	var dest *domain.Location
	if req.Origin != nil {
		origin = &domain.Location{Lat: req.Origin.Lat, Lng: req.Origin.Lng}
	}
	if req.Destination != nil {
		dest = &domain.Location{Lat: req.Destination.Lat, Lng: req.Destination.Lng}
	}
	order, err := s.svc.AdminUpdateOrder(ctx, req.OrderID, origin, dest)
	if err != nil {
		return nil, mapServiceError(err)
	}
	resp := transport.FromOrder(order)
	return &resp, nil
}

func (s *Server) AdminListDrones(ctx context.Context, _ *Empty) (*ListDronesResponse, error) {
	if _, err := requireRole(ctx, domain.RoleAdmin); err != nil {
		return nil, err
	}
	drones, err := s.svc.AdminListDrones(ctx)
	if err != nil {
		return nil, mapServiceError(err)
	}
	resp := &ListDronesResponse{Drones: make([]transport.DroneResponse, 0, len(drones))}
	for _, drone := range drones {
		resp.Drones = append(resp.Drones, transport.FromDrone(drone))
	}
	return resp, nil
}

func (s *Server) AdminMarkDroneBroken(ctx context.Context, req *DroneIDRequest) (*transport.DroneResponse, error) {
	if _, err := requireRole(ctx, domain.RoleAdmin); err != nil {
		return nil, err
	}
	drone, err := s.svc.AdminMarkDroneBroken(ctx, req.DroneID)
	if err != nil {
		return nil, mapServiceError(err)
	}
	resp := transport.FromDrone(drone)
	return &resp, nil
}

func (s *Server) AdminMarkDroneFixed(ctx context.Context, req *DroneIDRequest) (*transport.DroneResponse, error) {
	if _, err := requireRole(ctx, domain.RoleAdmin); err != nil {
		return nil, err
	}
	drone, err := s.svc.AdminMarkDroneFixed(ctx, req.DroneID)
	if err != nil {
		return nil, mapServiceError(err)
	}
	resp := transport.FromDrone(drone)
	return &resp, nil
}

