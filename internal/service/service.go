package service

import (
	"database/sql"
	"github.com/sirupsen/logrus"
	"github.com/zhukovrost/pasteAPI/internal/repository"
	"github.com/zhukovrost/pasteAPI/pkg/cache"
	"github.com/zhukovrost/pasteAPI/pkg/rabbitmq"
	"sync"
)

type Service struct {
	*Config
	Models *repository.Models
	Deps   *Dependencies
	Wg     *sync.WaitGroup
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
	DB     *sql.DB
	Mailer *rabbitmq.Connection
	Redis  *cache.MyCache
}

func New(cfg *Config, deps *Dependencies, models *repository.Models) *Service {
	return &Service{
		Config: cfg,
		Deps:   deps,
		Models: models,
		Wg:     &sync.WaitGroup{},
	}
}

func (s *Service) Background(fn func()) {
	s.Wg.Add(1)
	go func() {
		defer s.Wg.Done()
		// Recover any panic.
		defer func() {
			if err := recover(); err != nil {
				s.Deps.Logger.Error(err)
			}
		}()

		fn()
	}()
}
