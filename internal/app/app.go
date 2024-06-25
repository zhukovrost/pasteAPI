package app

import (
	"flag"
	"pasteAPI/internal/config"
	"pasteAPI/internal/http/v1"
	"pasteAPI/internal/metrics"
	"pasteAPI/internal/postgres"
	"pasteAPI/internal/repository"
	"pasteAPI/internal/server"
	"pasteAPI/internal/service"
	"pasteAPI/pkg/logger"
	"pasteAPI/pkg/mailer"
)

func Run(cfg *config.Config) {
	var needDebug bool
	flag.BoolVar(&needDebug, "debug", false, "turns on debug level (log)")
	flag.Parse()

	log := logger.New(needDebug)

	mailer := mailer.New(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.Sender)

	db, err := postgres.OpenDB(cfg)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()
	log.Info("database connection pool established")

	metrics.PostMetrics(db.Stats())

	service := service.New(cfg, log, mailer)
	models := repository.NewModels(db)

	handler := v1.NewHandler(service, models)
	srv := server.New(cfg, handler)

	if err = server.Run(srv, service); err != nil {
		log.Fatal(err)
	}
}
