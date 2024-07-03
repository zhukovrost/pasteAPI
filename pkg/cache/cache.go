package cache

import (
	"github.com/redis/go-redis/v9"
	"sync"
	"time"
)

type MyCache struct {
	Cache          *redis.Client
	DefaultTimeout time.Duration
	Expiration     time.Duration
}

var (
	once   sync.Once
	client *MyCache
)

func New(addr, password string, db int, timeout, expiration string) *MyCache {
	once.Do(func() {
		cache := redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		})

		defaultTimeout := 15 * time.Second

		if timeout != "" {
			newTimeout, err := time.ParseDuration(timeout)
			if err != nil {
				panic(err)
			}
			defaultTimeout = newTimeout
		}

		defaultExpiration := 15 * time.Minute

		if expiration != "" {
			newExp, err := time.ParseDuration(expiration)
			if err != nil {
				panic(err)
			}
			defaultExpiration = newExp
		}

		client = &MyCache{Cache: cache, DefaultTimeout: defaultTimeout, Expiration: defaultExpiration}
	})

	return client
}

func (c *MyCache) CloseConn() error {
	return c.Cache.Close()
}
