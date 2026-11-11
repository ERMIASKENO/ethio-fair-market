package main

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// User is the public user record.
type User struct {
	ID          string    `json:"id"`
	Phone       string    `json:"phone"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
}

// OTP is a stored one-time password.
type OTP struct {
	ID        int64
	CodeHash  string
	Salt      string
	Attempts  int
	ExpiresAt time.Time
}

// Store is the persistence interface (Postgres in production, fake in tests).
type Store interface {
	CountRecentOTPs(ctx context.Context, phone string, since time.Time) (int, error)
	InsertOTP(ctx context.Context, phone, hash, salt string, expires time.Time) error
	LatestOTP(ctx context.Context, phone string) (*OTP, error)
	IncrementAttempts(ctx context.Context, id int64) error
	ConsumeOTP(ctx context.Context, id int64) error
	UpsertUser(ctx context.Context, phone string) (*User, error)
	GetUser(ctx context.Context, id string) (*User, error)
	UpdateDisplayName(ctx context.Context, id, name string) (*User, error)
}

var errNotFound = errors.New("not found")

type pgStore struct{ db *sql.DB }

func (s *pgStore) CountRecentOTPs(ctx context.Context, phone string, since time.Time) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM otp_codes WHERE phone=$1 AND created_at > $2`, phone, since).Scan(&n)
	return n, err
}

func (s *pgStore) InsertOTP(ctx context.Context, phone, hash, salt string, expires time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO otp_codes (phone, code_hash, salt, expires_at) VALUES ($1,$2,$3,$4)`, phone, hash, salt, expires)
	return err
}

func (s *pgStore) LatestOTP(ctx context.Context, phone string) (*OTP, error) {
	var o OTP
	err := s.db.QueryRowContext(ctx, `SELECT id, code_hash, salt, attempts, expires_at FROM otp_codes
		WHERE phone=$1 AND consumed_at IS NULL ORDER BY created_at DESC LIMIT 1`, phone).
		Scan(&o.ID, &o.CodeHash, &o.Salt, &o.Attempts, &o.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errNotFound
	}
	return &o, err
}

func (s *pgStore) IncrementAttempts(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE otp_codes SET attempts = attempts + 1 WHERE id=$1`, id)
	return err
}

func (s *pgStore) ConsumeOTP(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE otp_codes SET consumed_at = now() WHERE id=$1`, id)
	return err
}

func (s *pgStore) UpsertUser(ctx context.Context, phone string) (*User, error) {
	var u User
	err := s.db.QueryRowContext(ctx, `INSERT INTO users (phone, last_login) VALUES ($1, now())
		ON CONFLICT (phone) DO UPDATE SET last_login = now()
		RETURNING id, phone, display_name, created_at`, phone).Scan(&u.ID, &u.Phone, &u.DisplayName, &u.CreatedAt)
	return &u, err
}

func (s *pgStore) GetUser(ctx context.Context, id string) (*User, error) {
	var u User
	err := s.db.QueryRowContext(ctx, `SELECT id, phone, display_name, created_at FROM users WHERE id=$1`, id).
		Scan(&u.ID, &u.Phone, &u.DisplayName, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errNotFound
	}
	return &u, err
}

func (s *pgStore) UpdateDisplayName(ctx context.Context, id, name string) (*User, error) {
	var u User
	err := s.db.QueryRowContext(ctx, `UPDATE users SET display_name=$2 WHERE id=$1 RETURNING id, phone, display_name, created_at`, id, name).
		Scan(&u.ID, &u.Phone, &u.DisplayName, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errNotFound
	}
	return &u, err
}
