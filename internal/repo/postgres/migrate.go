package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ApplyMigrations(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	if err := ensureMigrationsTable(ctx, pool); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".sql") {
			files = append(files, filepath.Join(dir, name))
		}
	}
	sort.Strings(files)
	for _, path := range files {
		version := filepath.Base(path)
		applied, err := isMigrationApplied(ctx, pool, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("apply %s: %w", version, err)
		}
		if err := markMigrationApplied(ctx, pool, version); err != nil {
			return err
		}
	}
	return nil
}

func ensureMigrationsTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version text PRIMARY KEY,
  applied_at timestamptz NOT NULL DEFAULT now()
)
`)
	return err
}

func isMigrationApplied(ctx context.Context, pool *pgxpool.Pool, version string) (bool, error) {
	row := pool.QueryRow(ctx, `SELECT 1 FROM schema_migrations WHERE version = $1`, version)
	var one int
	err := row.Scan(&one)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return false, err
}

func markMigrationApplied(ctx context.Context, pool *pgxpool.Pool, version string) error {
	_, err := pool.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version)
	return err
}
