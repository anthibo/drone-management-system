package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"penny-assesment/internal/config"
	"penny-assesment/internal/events"
	natspub "penny-assesment/internal/events/nats"
	"penny-assesment/internal/repo/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.LoadWorker()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	if !cfg.OutboxEnabled {
		log.Printf("outbox disabled; exiting")
		return
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

	publisher, err := natspub.New(cfg.NATSURL, cfg.NATSSubject)
	if err != nil {
		log.Fatalf("nats error: %v", err)
	}
	defer publisher.Close()

	store := postgres.NewStore(pool)
	worker := &events.OutboxWorker{
		Repo:         store,
		Publisher:    publisher,
		PollInterval: cfg.OutboxInterval,
		BatchSize:    cfg.OutboxBatch,
	}

	log.Printf("outbox worker running (interval=%s batch=%d)", cfg.OutboxInterval, cfg.OutboxBatch)
	if err := worker.Start(ctx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		log.Fatalf("worker error: %v", err)
	}
}
