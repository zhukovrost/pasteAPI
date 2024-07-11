package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"sync"
	"time"
)

type MyCache struct {
	client *redis.Client
	Config
}

type Cache interface {
	Get(key string, result interface{}) error
	Set(key string, value interface{}) error
	Delete(key string) error
	Exists(key string) (bool, error)
	Invalidate(key string) error
	Close() error
	RateLimiter(ip string) (int64, error)
}

var (
	once    sync.Once
	myCache *MyCache
)

type Config struct {
	Addr       string
	Password   string
	DB         int
	Expiration time.Duration
}

func New(c Config) *MyCache {
	once.Do(func() {
		newClient := redis.NewClient(&redis.Options{
			Addr:     c.Addr,
			Password: c.Password,
			DB:       c.DB,
		})

		myCache = &MyCache{client: newClient, Config: c}
	})

	return myCache
}

func (c *MyCache) Get(key string, result interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}

	return json.Unmarshal(data, result)
}

func (c *MyCache) Set(key string, value interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.Expiration).Err()
}

func (c *MyCache) Exists(key string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return exists == 1, nil
}

func (c *MyCache) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.client.Del(ctx, key).Err()
}

func (c *MyCache) Invalidate(pattern string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}

	// Using Cache pipeline to delete keys atomically
	pipe := c.client.Pipeline()
	for _, key := range keys {
		pipe.Del(ctx, key)
	}

	_, err = pipe.Exec(ctx)
	return err
}

func (c *MyCache) Close() error {
	return c.client.Close()
}

func (c *MyCache) RateLimiter(ip string) (int64, error) {
	redisKey := fmt.Sprintf("client:%s", ip)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipe := c.client.TxPipeline()
	incr := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, time.Second)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return incr.Result()
}
