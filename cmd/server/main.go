package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"google.golang.org/grpc"

	"penny-assesment/internal/auth"
	"penny-assesment/internal/config"
	"penny-assesment/internal/events"
	natspub "penny-assesment/internal/events/nats"
	"penny-assesment/internal/repo/postgres"
	"penny-assesment/internal/service"
	"penny-assesment/internal/transport/grpcapi"
	"penny-assesment/internal/transport/httpapi"
	"penny-assesment/internal/transport/thriftapi"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db error: %v", err)
	}
	defer pool.Close()

	if cfg.MigrateOnStart {
		if err := postgres.ApplyMigrations(ctx, pool, "migrations"); err != nil {
			log.Fatalf("migration error: %v", err)
		}
	}

	store := postgres.NewStore(pool)
	svc := service.New(store, cfg.DroneSpeedMPS)
	authenticator := auth.New(cfg.JWTSecret, cfg.JWTTTL)

	var publisher events.Publisher = events.NoopPublisher{}
	if cfg.OutboxEnabled {
		natsPublisher, err := natspub.New(cfg.NATSURL, cfg.NATSSubject)
		if err != nil {
			log.Fatalf("nats error: %v", err)
		}
		publisher = natsPublisher
		defer publisher.Close()
	}

	httpHandler := httpapi.NewServer(svc, authenticator)
	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpHandler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	grpcServer := grpcapi.NewServer(svc, authenticator)
	grpcListener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("grpc listen error: %v", err)
	}

	thriftServer, err := thriftapi.NewServer(cfg.ThriftAddr, svc, authenticator)
	if err != nil {
		log.Fatalf("thrift error: %v", err)
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		log.Printf("http listening on %s", cfg.HTTPAddr)
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	g.Go(func() error {
		log.Printf("grpc listening on %s", cfg.GRPCAddr)
		err := grpcServer.Serve(grpcListener)
		if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			return err
		}
		return nil
	})

	g.Go(func() error {
		log.Printf("thrift listening on %s", cfg.ThriftAddr)
		return thriftServer.Serve()
	})

	if cfg.OutboxEnabled {
		worker := &events.OutboxWorker{
			Repo:         store,
			Publisher:    publisher,
			PollInterval: cfg.OutboxInterval,
			BatchSize:    cfg.OutboxBatch,
		}
		g.Go(func() error {
			log.Printf("outbox worker running (interval=%s batch=%d)", cfg.OutboxInterval, cfg.OutboxBatch)
			err := worker.Start(ctx)
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return err
		})
	}

	g.Go(func() error {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
		grpcServer.GracefulStop()
		thriftServer.Stop()
		return nil
	})

	if err := g.Wait(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
