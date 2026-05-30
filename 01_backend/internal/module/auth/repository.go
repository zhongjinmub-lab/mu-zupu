package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidCredentials = errors.New("invalid email or password")

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return Repository{DB: db}
}

func (r Repository) CreateUser(ctx context.Context, email, passwordHash, nickname string) (User, error) {
	const q = `
INSERT INTO users(email, password_hash, nickname)
VALUES ($1, $2, NULLIF($3, ''))
RETURNING id::text, email::text, COALESCE(nickname, ''), status, created_at, updated_at, last_login_at`
	var u User
	err := r.DB.QueryRow(ctx, q, email, passwordHash, nickname).Scan(
		&u.ID, &u.Email, &u.Nickname, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.LastLogin,
	)
	return u, err
}

func (r Repository) FindByEmail(ctx context.Context, email string) (User, string, error) {
	const q = `
SELECT id::text, email::text, COALESCE(nickname, ''), status, created_at, updated_at, last_login_at, COALESCE(password_hash, '')
FROM users
WHERE email = $1 AND deleted_at IS NULL`
	var u User
	var passwordHash string
	err := r.DB.QueryRow(ctx, q, email).Scan(
		&u.ID, &u.Email, &u.Nickname, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.LastLogin, &passwordHash,
	)
	if err == pgx.ErrNoRows {
		return User{}, "", ErrInvalidCredentials
	}
	return u, passwordHash, err
}

func (r Repository) FindByID(ctx context.Context, userID string) (User, error) {
	const q = `
SELECT id::text, email::text, COALESCE(nickname, ''), status, created_at, updated_at, last_login_at
FROM users
WHERE id = $1 AND deleted_at IS NULL`
	var u User
	err := r.DB.QueryRow(ctx, q, userID).Scan(
		&u.ID, &u.Email, &u.Nickname, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.LastLogin,
	)
	return u, err
}

func (r Repository) TouchLastLogin(ctx context.Context, userID string) error {
	const q = `UPDATE users SET last_login_at = $2, updated_at = $2 WHERE id = $1`
	_, err := r.DB.Exec(ctx, q, userID, time.Now().UTC())
	return err
}
