package grpcapi

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"penny-assesment/internal/auth"
	"penny-assesment/internal/domain"
	"penny-assesment/internal/transport"
)

func getClaims(ctx context.Context) (*auth.Claims, error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing token")
	}
	return claims, nil
}

func requireRole(ctx context.Context, role string) (*auth.Claims, error) {
	claims, err := getClaims(ctx)
	if err != nil {
		return nil, err
	}
	if claims.Role != role {
		return nil, status.Error(codes.PermissionDenied, "forbidden")
	}
	return claims, nil
}

func mapServiceError(err error) error {
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		return status.Error(codes.Unauthenticated, "unauthorized")
	case errors.Is(err, domain.ErrForbidden):
		return status.Error(codes.PermissionDenied, "forbidden")
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, "not found")
	case errors.Is(err, domain.ErrConflict):
		return status.Error(codes.Aborted, "conflict")
	case errors.Is(err, domain.ErrInvalid):
		return status.Error(codes.InvalidArgument, "invalid request")
	case errors.Is(err, domain.ErrPrecondition):
		return status.Error(codes.FailedPrecondition, "precondition failed")
	case errors.Is(err, domain.ErrNoJob):
		return status.Error(codes.NotFound, "no job available")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

func toDomainLocation(loc transport.Location) domain.Location {
	return domain.Location{Lat: loc.Lat, Lng: loc.Lng}
}
