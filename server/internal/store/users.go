package store

import (
	"context"
	"fmt"

	"github.com/astrazstudio/pushnotify/server/internal/models"
)

// CreateUser inserts a new user and returns it.
func (s *Store) CreateUser(ctx context.Context, email, fullName, passwordHash string) (*models.User, error) {
	var u models.User
	err := s.db.QueryRow(ctx, `
		INSERT INTO users (email, full_name, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, email, full_name, password_hash, is_active, created_at, updated_at`,
		email, fullName, passwordHash,
	).Scan(&u.ID, &u.Email, &u.FullName, &u.PasswordHash, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", mapErr(err))
	}
	return &u, nil
}

// GetUserByEmail looks up a user by email.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	err := s.db.QueryRow(ctx, `
		SELECT id, email, full_name, password_hash, is_active, created_at, updated_at
		FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.FullName, &u.PasswordHash, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &u, nil
}

// GetUserByID looks up a user by id.
func (s *Store) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	var u models.User
	err := s.db.QueryRow(ctx, `
		SELECT id, email, full_name, password_hash, is_active, created_at, updated_at
		FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.FullName, &u.PasswordHash, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &u, nil
}
