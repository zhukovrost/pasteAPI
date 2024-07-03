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
	"github.com/zhukovrost/pasteAPI/pkg/mailer"
	"github.com/zhukovrost/pasteAPI/pkg/postgres"
)

func Run(cfg *config.Config) {
	log := logger.New(config.NeedDebug)

	mailer := mailer.New(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.Sender)

	//addr := cfg.Addr + ":" + cfg.Redis.Port
	addr := "localhost:6379"
	cache := cache.New(addr, cfg.Redis.Password, cfg.Redis.DB, cfg.Redis.Timeout, cfg.Redis.Expiration)
	defer cache.CloseConn()

	db, err := postgres.OpenDB(cfg.DB.DSN, cfg.DB.MaxIdleTime, cfg.DB.MaxOpenConns, cfg.DB.MaxIdleConns)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()
	log.Info("database connection pool established")

	metrics.PostMetrics(db.Stats())

	service := service.New(cfg, log, mailer, cache)
	models := repository.NewModels(db)

	handler := v1.NewHandler(service, models)
	srv := server.New(cfg, handler)

	if err = server.Run(srv, service); err != nil {
		log.Fatal(err)
	}
}
