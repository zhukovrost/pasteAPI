package v1

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/zhukovrost/pasteAPI/internal/repository/models"
	"github.com/zhukovrost/pasteAPI/pkg/cache"
	"github.com/zhukovrost/pasteAPI/pkg/validator"
)

func getFromCache(myCache *cache.MyCache, key string, result interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), myCache.DefaultTimeout)
	defer cancel()

	data, err := myCache.Cache.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, result); err != nil {
		return err
	}

	v := validator.New()
	switch result.(type) {
	case *models.Paste:
		models.ValidateTime(v, result.(*models.Paste))
	case *ListPastesOutput:
		for _, paste := range result.(*ListPastesOutput).Pastes {
			models.ValidateTime(v, paste)
		}
	default:
		return errors.New("data processing error: invalid result type")
	}

	if !v.Valid() {
		return errors.New("cache error: some pastes are expired")
	}

	return nil
}

func setCache(myCache *cache.MyCache, key string, value interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), myCache.DefaultTimeout)
	defer cancel()

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	err = myCache.Cache.Set(ctx, key, data, myCache.Expiration).Err()
	if err != nil {
		return err
	}

	return nil
}

func existsInCache(myCache *cache.MyCache, key string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), myCache.DefaultTimeout)
	defer cancel()

	exists, err := myCache.Cache.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return exists == 1, nil
}

func deleteFromCache(myCache *cache.MyCache, key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), myCache.DefaultTimeout)
	defer cancel()

	return myCache.Cache.Del(ctx, key).Err()
}

func invalidateCache(myCache *cache.MyCache, pattern string) error {
	ctx, cancel := context.WithTimeout(context.Background(), myCache.DefaultTimeout)
	defer cancel()

	keys, err := myCache.Cache.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}

	// Using Redis pipeline to delete keys atomically
	pipe := myCache.Cache.Pipeline()
	for _, key := range keys {
		pipe.Del(ctx, key)
	}

	_, err = pipe.Exec(ctx)
	return err
}
