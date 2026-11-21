package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Client wraps a go-redis client with health check and lifecycle methods.
type Client struct {
	Raw *goredis.Client
}

// New initializes a Redis client and verifies connectivity with PING.
func New(ctx context.Context, addr, password string, db int) (*Client, error) {
	rdb := goredis.NewClient(&goredis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &Client{Raw: rdb}, nil
}

// HealthCheck verifies Redis is reachable.
func (c *Client) HealthCheck(ctx context.Context) error {
	return c.Raw.Ping(ctx).Err()
}

// Close shuts down the Redis connection pool.
func (c *Client) Close() error {
	if c == nil || c.Raw == nil {
		return nil
	}
	return c.Raw.Close()
}
