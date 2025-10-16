package clickhouse

import (
	"context"
	"fmt"
	"log/slog"

	click "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
}

type Client struct {
	conn driver.Conn
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	conn, err := click.Open(&click.Options{
		Addr: []string{fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)},
		Auth: click.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
		Debug: true,
		Debugf: func(format string, v ...any) {
			slog.Debug(format, v...)
		},
	})
	if err != nil {
		return nil, err
	}

	cl := Client{
		conn: conn,
	}

	if err := conn.Ping(ctx); err != nil {
		if exception, ok := err.(*click.Exception); ok {
			slog.Error("exception caught", slog.Int64("code", int64(exception.Code)), slog.String("message", exception.Message), slog.String("stack_trace", exception.StackTrace))
		}
		return nil, err
	}

	return &cl, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Query(ctx context.Context, query string, args ...interface{}) (driver.Rows, error) {
	return c.conn.Query(ctx, query, args...)
}

func (c *Client) QueryRow(ctx context.Context, query string, args ...interface{}) driver.Row {
	return c.conn.QueryRow(ctx, query, args...)
}
