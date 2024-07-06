package postgres

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

type Config struct {
	DSN          string
	MaxIdleTime  time.Duration
	MaxOpenConns int
	MaxIdleConns int
}

// OpenDB function returns a sql.DB connection pool.
func OpenDB(c Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", c.DSN)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxIdleTime(c.MaxIdleTime)
	db.SetMaxOpenConns(c.MaxOpenConns)
	db.SetMaxIdleConns(c.MaxIdleConns)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}
