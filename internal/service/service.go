package service

import (
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/zhukovrost/pasteAPI/internal/repository"
	"github.com/zhukovrost/pasteAPI/internal/repository/models"
	"github.com/zhukovrost/pasteAPI/pkg/cache"
	"github.com/zhukovrost/pasteAPI/pkg/postgres"
	"github.com/zhukovrost/pasteAPI/pkg/rabbitmq"
	"sync"
)

type Services struct {
	Config
	Deps   Dependencies
	Pastes Pastes
}

type Pastes interface {
	GetList(settings SearchSettings, user *models.User) (*ListPastesOutput, error)
	GetPaste(id uint16, user *models.User) (*models.Paste, error)
	Delete(id uint16) error
	Create(paste *models.Paste, creator *models.User) error
	GetPasteForUpdate(pasteId uint16, in UpdatePasteInput) (*models.Paste, error)
	Update(paste *models.Paste) error
	GivePermission(pasteId uint16, userId int64) (*PastePermissionResponse, error)
}

type Config struct {
	Host           string
	Port           int
	Env            string
	Status         string
	ActivationLink string
	ResetLink      string
	Limiter        struct {
		RPS     float64
		Burst   int
		Enabled bool
	}
	CORS struct {
		TrustedOrigins []string
	}
	BuildTime string
	Version   string
}

type Dependencies struct {
	Logger *logrus.Logger
	DB     postgres.Database
	Mailer *rabbitmq.Connection
	Redis  *cache.MyCache
	Models *repository.Models
	Wg     *sync.WaitGroup
}

func New(cfg Config, deps Dependencies) *Services {
	return &Services{
		Config: cfg,
		Deps:   deps,
		Pastes: NewPasteService(deps.Models.Pastes, deps.Models.Permissions, deps.Logger, deps.Redis, deps.Wg),
	}
}

type SearchSettings struct {
	Title    string
	Category uint8
	Filters  models.Filters
}

type ListPastesOutput struct {
	Pastes   []*models.Paste  `json:"pastes"`
	Metadata *models.Metadata `json:"metadata"`
}

type PasteResp struct {
	R *models.Paste `json:"paste"`
}

type CreatePasteInput struct {
	Title    string `json:"title"`
	Category uint8  `json:"category,omitempty"`
	Text     string `json:"text"`
	Minutes  int32  `json:"minutes"`
}

type UpdatePasteInput struct {
	Title    *string `json:"title"`
	Category *uint8  `json:"category,omitempty"`
	Text     *string `json:"text"`
	Minutes  *int32  `json:"minutes"`
}

type PastePermissionResponse struct {
	Permission models.Permission `json:"permission"`
}

func getFromCache(myCache *cache.MyCache, key string, result interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), myCache.Timeout)
	defer cancel()

	data, err := myCache.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}

	return json.Unmarshal(data, result)
}

func setCache(myCache *cache.MyCache, key string, value interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), myCache.Timeout)
	defer cancel()

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return myCache.Set(ctx, key, data, myCache.Expiration).Err()
}

func existsInCache(myCache *cache.MyCache, key string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), myCache.Timeout)
	defer cancel()

	exists, err := myCache.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return exists == 1, nil
}

func deleteFromCache(myCache *cache.MyCache, key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), myCache.Timeout)
	defer cancel()

	return myCache.Del(ctx, key).Err()
}

func invalidateCache(myCache *cache.MyCache, pattern string) error {
	ctx, cancel := context.WithTimeout(context.Background(), myCache.Timeout)
	defer cancel()

	keys, err := myCache.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}

	// Using Redis pipeline to delete keys atomically
	pipe := myCache.Pipeline()
	for _, key := range keys {
		pipe.Del(ctx, key)
	}

	_, err = pipe.Exec(ctx)
	return err
}
