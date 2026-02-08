package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"penny-assesment/internal/auth"
	"penny-assesment/internal/domain"
	"penny-assesment/internal/service"
	"penny-assesment/internal/transport"
)

type Server struct {
	svc  *service.Service
	auth *auth.Authenticator
}

func NewServer(svc *service.Service, authenticator *auth.Authenticator) http.Handler {
	s := &Server{svc: svc, auth: authenticator}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)

	r.Post("/auth/token", s.handleIssueToken)

	r.Route("/drone", func(r chi.Router) {
		r.Use(s.requireRole(domain.RoleDrone))
		r.Post("/jobs/reserve", s.handleDroneReserve)
		r.Post("/orders/{id}/pickup", s.handleDronePickup)
		r.Post("/orders/{id}/deliver", s.handleDroneDeliver)
		r.Post("/orders/{id}/fail", s.handleDroneFail)
		r.Post("/broken", s.handleDroneBroken)
		r.Post("/heartbeat", s.handleDroneHeartbeat)
		r.Get("/orders/current", s.handleDroneCurrentOrder)
	})

	r.Route("/orders", func(r chi.Router) {
		r.Use(s.requireRole(domain.RoleEndUser))
		r.Post("/", s.handleSubmitOrder)
		r.Post("/{id}/withdraw", s.handleWithdrawOrder)
		r.Get("/{id}", s.handleGetOrder)
	})

	r.Route("/admin", func(r chi.Router) {
		r.Use(s.requireRole(domain.RoleAdmin))
		r.Get("/orders", s.handleAdminListOrders)
		r.Patch("/orders/{id}", s.handleAdminUpdateOrder)
		r.Get("/drones", s.handleAdminListDrones)
		r.Post("/drones/{id}/broken", s.handleAdminDroneBroken)
		r.Post("/drones/{id}/fixed", s.handleAdminDroneFixed)
	})

	return r
}

func (s *Server) requireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := auth.ExtractBearerToken(r.Header.Get("Authorization"))
			if token == "" {
				writeError(w, domain.ErrUnauthorized)
				return
			}
			claims, err := s.auth.ParseToken(token)
			if err != nil {
				writeError(w, domain.ErrUnauthorized)
				return
			}
			if claims.Role != role {
				writeError(w, domain.ErrForbidden)
				return
			}
			ctx := auth.ContextWithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (s *Server) handleIssueToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, domain.ErrInvalid)
		return
	}
	if req.Name == "" || !domain.ValidateRole(req.Role) {
		writeError(w, domain.ErrInvalid)
		return
	}
	token, exp, err := s.auth.IssueToken(req.Name, req.Role)
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"token":      token,
		"expires_at": exp,
	})
}

func (s *Server) handleSubmitOrder(w http.ResponseWriter, r *http.Request) {
	claims := mustClaims(r)
	var req struct {
		Origin      transport.Location `json:"origin"`
		Destination transport.Location `json:"destination"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, domain.ErrInvalid)
		return
	}
	order, err := s.svc.SubmitOrder(r.Context(), claims.Subject, toDomainLocation(req.Origin), toDomainLocation(req.Destination))
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, transport.FromOrder(order))
}

func (s *Server) handleWithdrawOrder(w http.ResponseWriter, r *http.Request) {
	claims := mustClaims(r)
	orderID := chi.URLParam(r, "id")
	order, err := s.svc.WithdrawOrder(r.Context(), claims.Subject, orderID)
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, transport.FromOrder(order))
}

func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	claims := mustClaims(r)
	orderID := chi.URLParam(r, "id")
	view, err := s.svc.GetOrderView(r.Context(), claims.Subject, claims.Role, orderID)
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, transport.FromOrderView(view))
}

func (s *Server) handleDroneReserve(w http.ResponseWriter, r *http.Request) {
	claims := mustClaims(r)
	order, err := s.svc.DroneReserveJob(r.Context(), claims.Subject)
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, transport.FromOrder(order))
}

func (s *Server) handleDronePickup(w http.ResponseWriter, r *http.Request) {
	claims := mustClaims(r)
	orderID := chi.URLParam(r, "id")
	order, err := s.svc.DronePickup(r.Context(), claims.Subject, orderID)
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, transport.FromOrder(order))
}

func (s *Server) handleDroneDeliver(w http.ResponseWriter, r *http.Request) {
	claims := mustClaims(r)
	orderID := chi.URLParam(r, "id")
	order, err := s.svc.DroneDeliver(r.Context(), claims.Subject, orderID)
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, transport.FromOrder(order))
}

func (s *Server) handleDroneFail(w http.ResponseWriter, r *http.Request) {
	claims := mustClaims(r)
	orderID := chi.URLParam(r, "id")
	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, domain.ErrInvalid)
		return
	}
	order, err := s.svc.DroneFail(r.Context(), claims.Subject, orderID, req.Reason)
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, transport.FromOrder(order))
}

func (s *Server) handleDroneBroken(w http.ResponseWriter, r *http.Request) {
	claims := mustClaims(r)
	drone, err := s.svc.DroneMarkBroken(r.Context(), claims.Subject)
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, transport.FromDrone(drone))
}

func (s *Server) handleDroneHeartbeat(w http.ResponseWriter, r *http.Request) {
	claims := mustClaims(r)
	var req struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, domain.ErrInvalid)
		return
	}
	view, err := s.svc.DroneHeartbeat(r.Context(), claims.Subject, domain.Location{Lat: req.Lat, Lng: req.Lng})
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, transport.FromDroneStatus(view))
}

func (s *Server) handleDroneCurrentOrder(w http.ResponseWriter, r *http.Request) {
	claims := mustClaims(r)
	view, err := s.svc.DroneCurrentOrder(r.Context(), claims.Subject)
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, transport.FromOrderView(view))
}

func (s *Server) handleAdminListOrders(w http.ResponseWriter, r *http.Request) {
	statusParam := r.URL.Query().Get("status")
	var status *domain.OrderStatus
	if statusParam != "" {
		st := domain.OrderStatus(statusParam)
		status = &st
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	views, err := s.svc.AdminListOrders(r.Context(), service.OrderFilter{Status: status, Limit: limit, Offset: offset})
	if err != nil {
		writeError(w, err)
		return
	}
	resp := make([]transport.OrderViewResponse, 0, len(views))
	for _, view := range views {
		resp = append(resp, transport.FromOrderView(view))
	}
	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminUpdateOrder(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	var req struct {
		Origin      *transport.Location `json:"origin"`
		Destination *transport.Location `json:"destination"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, domain.ErrInvalid)
		return
	}
	var origin *domain.Location
	var dest *domain.Location
	if req.Origin != nil {
		origin = &domain.Location{Lat: req.Origin.Lat, Lng: req.Origin.Lng}
	}
	if req.Destination != nil {
		dest = &domain.Location{Lat: req.Destination.Lat, Lng: req.Destination.Lng}
	}
	order, err := s.svc.AdminUpdateOrder(r.Context(), orderID, origin, dest)
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, transport.FromOrder(order))
}

func (s *Server) handleAdminListDrones(w http.ResponseWriter, r *http.Request) {
	drones, err := s.svc.AdminListDrones(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	resp := make([]transport.DroneResponse, 0, len(drones))
	for _, drone := range drones {
		resp = append(resp, transport.FromDrone(drone))
	}
	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminDroneBroken(w http.ResponseWriter, r *http.Request) {
	droneID := chi.URLParam(r, "id")
	drone, err := s.svc.AdminMarkDroneBroken(r.Context(), droneID)
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, transport.FromDrone(drone))
}

func (s *Server) handleAdminDroneFixed(w http.ResponseWriter, r *http.Request) {
	droneID := chi.URLParam(r, "id")
	drone, err := s.svc.AdminMarkDroneFixed(r.Context(), droneID)
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, transport.FromDrone(drone))
}

func mustClaims(r *http.Request) *auth.Claims {
	claims, _ := auth.ClaimsFromContext(r.Context())
	return claims
}

func toDomainLocation(loc transport.Location) domain.Location {
	return domain.Location{Lat: loc.Lat, Lng: loc.Lng}
}

