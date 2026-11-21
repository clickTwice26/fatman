package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Client wraps a pgxpool.Pool with lifecycle and health check helpers.
type Client struct {
	Pool *pgxpool.Pool
}

// New connects to PostgreSQL and verifies initial connectivity.
func New(ctx context.Context, databaseURL string) (*Client, error) {
	poolCfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}
	poolCfg.MaxConns = 25
	poolCfg.MinConns = 2
	poolCfg.MaxConnLifetime = 30 * time.Minute
	poolCfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &Client{Pool: pool}, nil
}

// HealthCheck verifies PostgreSQL is reachable.
func (c *Client) HealthCheck(ctx context.Context) error {
	return c.Pool.Ping(ctx)
}

// Close releases all pool connections.
func (c *Client) Close() {
	if c != nil && c.Pool != nil {
		c.Pool.Close()
	}
}
