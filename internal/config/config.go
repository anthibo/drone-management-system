package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL   string
	JWTSecret     string
	JWTTTL        time.Duration
	HTTPAddr      string
	GRPCAddr      string
	ThriftAddr    string
	DroneSpeedMPS float64
	MigrateOnStart bool
	NATSURL        string
	NATSSubject    string
	OutboxEnabled  bool
	OutboxInterval time.Duration
	OutboxBatch    int
}

func Load() (Config, error) {
	return load(true)
}

func LoadWorker() (Config, error) {
	return load(false)
}

func load(requireJWT bool) (Config, error) {
	var cfg Config
	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is required")
	}
	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if requireJWT && cfg.JWTSecret == "" {
		return cfg, fmt.Errorf("JWT_SECRET is required")
	}
	cfg.JWTTTL = getDuration("JWT_TTL", time.Hour)
	cfg.HTTPAddr = getString("HTTP_ADDR", ":8080")
	cfg.GRPCAddr = getString("GRPC_ADDR", ":9090")
	cfg.ThriftAddr = getString("THRIFT_ADDR", ":9091")
	cfg.DroneSpeedMPS = getFloat("DRONE_SPEED_MPS", 15.0)
	cfg.MigrateOnStart = getBool("MIGRATE_ON_START", true)
	cfg.NATSURL = getString("NATS_URL", "nats://127.0.0.1:4222")
	cfg.NATSSubject = getString("NATS_SUBJECT", "drone.events")
	cfg.OutboxEnabled = getBool("OUTBOX_ENABLED", true)
	cfg.OutboxInterval = getDuration("OUTBOX_POLL_INTERVAL", time.Second)
	cfg.OutboxBatch = getInt("OUTBOX_BATCH_SIZE", 50)
	return cfg, nil
}

func getString(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func getDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func getFloat(key string, def float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

func getBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func getInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}
