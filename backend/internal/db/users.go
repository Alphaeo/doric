package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID      string
	Email   string
	Name    string
	Picture string
}

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(ctx context.Context) (*UserRepo, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://doric:doric@localhost:5432/doric"
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	return &UserRepo{pool: pool}, nil
}

func (r *UserRepo) UpsertUser(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (id, email, name, picture)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE
		SET email = $2, name = $3, picture = $4
	`
	_, err := r.pool.Exec(ctx, query, user.ID, user.Email, user.Name, user.Picture)
	return err
}

func (r *UserRepo) GetUser(ctx context.Context, id string) (*User, error) {
	var user User
	err := r.pool.QueryRow(ctx, "SELECT id, email, name, picture FROM users WHERE id = $1", id).
		Scan(&user.ID, &user.Email, &user.Name, &user.Picture)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
