package service

import (
	"github.com/sirupsen/logrus"
	"pasteAPI/internal/config"
	"pasteAPI/pkg/mailer"
	"sync"
)

type Service struct {
	Config *config.Config
	Logger *logrus.Logger
	Mailer *mailer.Mailer
	Wg     sync.WaitGroup
}

func New(cfg *config.Config, logger *logrus.Logger, mailer *mailer.Mailer) *Service {
	return &Service{
		Config: cfg,
		Logger: logger,
		Mailer: mailer,
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
