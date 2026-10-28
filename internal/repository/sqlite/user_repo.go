package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/safari3308/adguard-portal/internal/domain"
)

type userRepo struct {
	db *DB
}

func NewUserRepo(db *DB) domain.UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Upsert(ctx context.Context, u *domain.User) error {
	query := `
	INSERT INTO users (id, email, name, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		email = excluded.email,
		name = excluded.name,
		updated_at = excluded.updated_at;
	`
	now := time.Now().UTC()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	u.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query, u.ID, u.Email, u.Name, u.CreatedAt, u.UpdatedAt)
	return err
}

func (r *userRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, email, name, created_at, updated_at FROM users WHERE id = ?;`
	row := r.db.QueryRowContext(ctx, query, id)

	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, email, name, created_at, updated_at FROM users WHERE email = ?;`
	row := r.db.QueryRowContext(ctx, query, email)

	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
