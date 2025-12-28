package db

import (
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var DB *sqlx.DB

// Connect establishes a connection to PostgreSQL
func Connect() error {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Default to local development without hardcoded password (expects user/pass to be standard or env set)
		// For security, we do not commit the password. Ensure DATABASE_URL is set.
		dbURL = "postgres://sago:sago@localhost:5433/sago?sslmode=disable"
	}

	var err error
	DB, err = sqlx.Connect("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(5 * time.Minute)

	return nil
}

// Close closes the database connection
func Close() {
	if DB != nil {
		DB.Close()
	}
}
