// Package db is clipfolio's Postgres access layer: embedded migrations plus a
// Store exposing one method per query the API needs. There is no ORM -
// queries are plain SQL against pgx, kept small enough that a generated
// query layer would add indirection without saving real effort.
package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a lookup by id/email/token finds nothing.
var ErrNotFound = errors.New("not found")

// Store is clipfolio's Postgres access layer.
type Store struct {
	pool *pgxpool.Pool
}

// Open connects to Postgres at databaseURL and verifies it's reachable.
func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Store{pool: pool}, nil
}

// Close releases the underlying connection pool.
func (s *Store) Close() {
	s.pool.Close()
}

// Migrate applies any pending schema migrations.
func (s *Store) Migrate(ctx context.Context) error {
	return Migrate(ctx, s.pool)
}

// Exec runs a raw statement against the pool. Exposed narrowly for test
// setup/teardown (truncating tables between test runs); application code
// should add a typed method instead of reaching for this.
func (s *Store) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := s.pool.Exec(ctx, sql, args...)
	return err
}

func wrapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
