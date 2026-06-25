// Package redis provides a redis client
package redis

import (
	"log/slog"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"

	"github.com/beihai0xff/turl/configs"
)

// Nil is the redis.Nil, used to check if a key exists
var Nil = redis.Nil

// Client returns a redis client
func Client(c *configs.RedisConfig) redis.UniversalClient {
	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:          c.Addr,
		DialTimeout:    c.DialTimeout,
		MaxIdleConns:   c.MaxConn,
		MaxActiveConns: c.MaxConn,
	})

	// Trace redis commands via OpenTelemetry; a no-op until a provider is set.
	if err := redisotel.InstrumentTracing(rdb); err != nil {
		slog.Warn("failed to instrument redis tracing", slog.Any("error", err))
	}

	return rdb
}
