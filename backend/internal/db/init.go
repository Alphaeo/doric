package db

import (
	"context"
	"fmt"
)

func (r *UserRepo) InitDB(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			name TEXT,
			picture TEXT
		);
	`
	_, err := r.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}
	return nil
}
