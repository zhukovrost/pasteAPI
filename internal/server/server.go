package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"pasteAPI/internal/config"
	"pasteAPI/internal/http"
	"pasteAPI/internal/http/v1"
	"pasteAPI/internal/service"
	"syscall"
	"time"
)

func New(config *config.Config, handler *v1.Handler) *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      router.NewRouter(handler),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}

// Run function runs the server with a graceful shutdown
func Run(server *http.Server, service *service.Service) error {
	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit
		service.Logger.WithFields(map[string]interface{}{
			"signal": s.String(),
		}).Info("caught signal")

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		err := server.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}
		service.Logger.Info("completing background tasks")
		service.Wg.Wait()
		shutdownError <- nil
	}()

	service.Logger.WithFields(map[string]interface{}{
		"addr": server.Addr,
		"env":  service.Config.Env,
	}).Info("starting server")

	err := server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError
	if err != nil {
		return err
	}

	service.Logger.WithFields(map[string]interface{}{
		"addr": server.Addr,
	}).Info("stopped server")

	return nil
}
