package app

import (
	"github.com/zhukovrost/pasteAPI/internal/config"
	"github.com/zhukovrost/pasteAPI/internal/http/v1"
	"github.com/zhukovrost/pasteAPI/internal/metrics"
	"github.com/zhukovrost/pasteAPI/internal/repository"
	"github.com/zhukovrost/pasteAPI/internal/server"
	"github.com/zhukovrost/pasteAPI/internal/service"
	"github.com/zhukovrost/pasteAPI/pkg/cache"
	"github.com/zhukovrost/pasteAPI/pkg/logger"
	"github.com/zhukovrost/pasteAPI/pkg/postgres"
	"github.com/zhukovrost/pasteAPI/pkg/rabbitmq"

	"time"
)

func Run(cfg *config.Config) {
	log := logger.New(cfg.Logger.NeedDebug)

	log.Info("configuring mailer (RabbitMQ)")

	mailerWaitTime, err := time.ParseDuration(cfg.RabbitMQ.WaitTime)
	if err != nil {
		mailerWaitTime = 5 * time.Second // default
	}

	mailerTimeout, err := time.ParseDuration(cfg.RabbitMQ.Timeout)
	if err != nil {
		mailerTimeout = 5 * time.Second // default
	}

	mailer, err := rabbitmq.New(rabbitmq.Config{
		URL:          cfg.RabbitMQ.URL,
		WaitTime:     mailerWaitTime,
		Timeout:      mailerTimeout,
		Attempts:     cfg.RabbitMQ.Attempts,
		Exchange:     cfg.RabbitMQ.Exchange,
		ExchangeType: cfg.RabbitMQ.ExchangeType,
		Queue:        cfg.RabbitMQ.Queue,
	})

	if err != nil {
		panic(err)
	}

	defer mailer.Close()

	log.Info("configuring cache (Redis)")

	cacheExpiration, err := time.ParseDuration(cfg.Redis.Expiration)
	if err != nil {
		cacheExpiration = 15 * time.Minute // default
	}

	cacheTimeout, err := time.ParseDuration(cfg.Redis.Timeout)
	if err != nil {
		cacheTimeout = 5 * time.Second // default
	}

	cache := cache.New(cache.Config{
		Addr:       cfg.Redis.Host + ":" + cfg.Redis.Port,
		Password:   cfg.Redis.Password,
		DB:         cfg.Redis.DB,
		Expiration: cacheExpiration,
		Timeout:    cacheTimeout,
	})
	defer cache.CloseConn()

	log.Info("configuring database (PostgreSQL)")

	idleConnsDur, err := time.ParseDuration(cfg.DB.MaxIdleTime)
	if err != nil {
		idleConnsDur = 10 * time.Minute // default
	}

	db, err := postgres.OpenDB(postgres.Config{
		DSN:          cfg.DB.DSN,
		MaxIdleTime:  idleConnsDur,
		MaxOpenConns: cfg.DB.MaxOpenConns,
		MaxIdleConns: cfg.DB.MaxIdleConns,
	})

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	metrics.PostMetrics(db.Stats())

	log.Info("service connections are established")

	service := service.New(
		&service.Config{
			Host:           cfg.Host,
			Port:           cfg.Port,
			Env:            cfg.Env,
			Status:         cfg.Status,
			ActivationLink: cfg.ActivationLink,
			ResetLink:      cfg.ResetLink,
			Limiter: struct {
				RPS     float64
				Burst   int
				Enabled bool
			}(cfg.Limiter),
			CORS:      struct{ TrustedOrigins []string }(cfg.CORS),
			BuildTime: config.BuildTime,
			Version:   config.Version,
		},
		&service.Dependencies{
			Logger: log,
			DB:     db,
			Mailer: mailer,
			Redis:  cache,
		},
		repository.NewModels(db),
	)

	handler := v1.NewHandler(service)
	srv := server.New(handler, service.Port)

	if err = server.Run(srv, service); err != nil {
		log.Fatal(err)
	}
}
