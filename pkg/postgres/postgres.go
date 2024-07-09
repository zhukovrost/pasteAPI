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

type Database interface {
	OpenDB(c Config) error
	Stats() sql.DBStats
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	Close() error
}

type Connection struct {
	*sql.DB
}

// OpenDB function returns a sql.DB connection pool.
func (conn *Connection) OpenDB(c Config) error {
	db, err := sql.Open("postgres", c.DSN)
	if err != nil {
		return err
	}

	db.SetConnMaxIdleTime(c.MaxIdleTime)
	db.SetMaxOpenConns(c.MaxOpenConns)
	db.SetMaxIdleConns(c.MaxIdleConns)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return err
	}

	conn.DB = db
	return nil
}
