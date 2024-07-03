package service

import (
	"github.com/sirupsen/logrus"
	"github.com/zhukovrost/pasteAPI/internal/config"
	"github.com/zhukovrost/pasteAPI/pkg/cache"
	"github.com/zhukovrost/pasteAPI/pkg/mailer"
	"sync"
)

type Service struct {
	Config *config.Config
	Logger *logrus.Logger
	Mailer *mailer.Mailer
	Redis  *cache.MyCache
	Wg     sync.WaitGroup
}

func New(cfg *config.Config, logger *logrus.Logger, mailer *mailer.Mailer, cache *cache.MyCache) *Service {
	return &Service{
		Config: cfg,
		Logger: logger,
		Mailer: mailer,
		Redis:  cache,
		Wg:     sync.WaitGroup{},
	}
}

func (s *Service) Background(fn func()) {
	// TODO: rabbitmq
	s.Wg.Add(1)
	go func() {
		defer s.Wg.Done()
		// Recover any panic.
		defer func() {
			if err := recover(); err != nil {
				s.Logger.Error(err)
			}
		}()

		fn()
	}()
}
