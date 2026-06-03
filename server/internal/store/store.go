// Package store centralizes all PostgreSQL access. Every query is
// parameterized; no SQL is built via string concatenation of user input.
package store

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")

// Store wraps the pgx connection pool and exposes repository methods.
type Store struct {
	db *pgxpool.Pool
}

// New constructs a Store.
func New(db *pgxpool.Pool) *Store { return &Store{db: db} }

// Pool exposes the underlying pool for advanced callers (e.g. transactions).
func (s *Store) Pool() *pgxpool.Pool { return s.db }

// mapErr normalizes pgx.ErrNoRows into our sentinel ErrNotFound.
func mapErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
