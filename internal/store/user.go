package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"ringbuffer/internal/models"
)

// ErrEmailTaken — email уже занят.
var ErrEmailTaken = errors.New("email уже занят")

const userColumns = "id, email, password, created_at, updated_at"

func scanUser(r scanner) (models.User, error) {
	var u models.User
	err := r.Scan(&u.ID, &u.Email, &u.Password, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return u, err
	}
	return u, nil
}

// CreateUser создаёт нового пользователя.
func (s *Store) CreateUser(ctx context.Context, in models.UserInput) (models.User, error) {
	u, err := scanUser(s.pool.QueryRow(ctx, `
		INSERT INTO users (email, password)
		VALUES ($1, $2)
		RETURNING `+userColumns,
		in.Email, in.Password))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique violation
			return u, ErrEmailTaken
		}
		return u, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// GetUserByEmail возвращает пользователя по email.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	u, err := scanUser(s.pool.QueryRow(ctx, `
		SELECT `+userColumns+` FROM users WHERE email = $1`, email))
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	if err != nil {
		return u, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

// GetUserByID возвращает пользователя по ID.
func (s *Store) GetUserByID(ctx context.Context, id int64) (models.User, error) {
	u, err := scanUser(s.pool.QueryRow(ctx, `
		SELECT `+userColumns+` FROM users WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	if err != nil {
		return u, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}
