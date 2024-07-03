package v1

import (
	"context"
	"encoding/json"
	"github.com/zhukovrost/pasteAPI/internal/repository/models"
	"github.com/zhukovrost/pasteAPI/pkg/cache"
)

func getFromCache(myCache *cache.MyCache, key string) (*models.Paste, error) {
	ctx, cancel := context.WithTimeout(context.Background(), myCache.DefaultTimeout)
	defer cancel()

	data, err := myCache.Cache.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	result := &models.Paste{}
	if err := json.Unmarshal(data, result); err != nil {
		return nil, err
	}

	return result, nil
}

func setCache(myCache *cache.MyCache, key string, model *models.Paste) error {
	ctx, cancel := context.WithTimeout(context.Background(), myCache.DefaultTimeout)
	defer cancel()

	data, err := json.Marshal(model)
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
