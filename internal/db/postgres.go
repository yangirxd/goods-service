package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewPostgres(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for {
		err := db.PingContext(ctx)
		if err == nil {
			break
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for postgres: %w", ctx.Err())
		default:
			fmt.Printf("Waiting for postgres... %v\n", err)
			time.Sleep(1 * time.Second)
		}
	}

	return db, nil
}
