package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func RunMigrations(db *sql.DB, migrationsDir string) error {
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("listing migration files: %w", err)
	}

	for _, file := range files {
		fmt.Printf("Running migration: %s\n", file)
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("reading migration file: %w", err)
		}

		if _, err := db.ExecContext(context.Background(), string(content)); err != nil {
			return fmt.Errorf("executing migration %s: %w", file, err)
		}
	}

	return nil
}
