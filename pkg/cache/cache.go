package cache

import (
	"github.com/redis/go-redis/v9"
	"sync"
	"time"
)

type MyCache struct {
	*redis.Client
	Config
}

var (
	once   sync.Once
	client *MyCache
)

type Config struct {
	Addr       string
	Password   string
	DB         int
	Expiration time.Duration
	Timeout    time.Duration
}

func New(c Config) *MyCache {
	once.Do(func() {
		cache := redis.NewClient(&redis.Options{
			Addr:     c.Addr,
			Password: c.Password,
			DB:       c.DB,
		})

		client = &MyCache{Client: cache, Config: c}
	})

	return client
}
