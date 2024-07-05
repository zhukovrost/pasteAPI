package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/zhukovrost/pasteAPI/internal/autoclean"
	"github.com/zhukovrost/pasteAPI/internal/http"
	"github.com/zhukovrost/pasteAPI/internal/http/v1"
	"github.com/zhukovrost/pasteAPI/internal/service"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func New(handler *v1.Handler, port int) *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      router.NewRouter(handler),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}

// Run function runs the server with a graceful shutdown
func Run(server *http.Server, service *service.Service) error {
	shutdownError := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go autoclean.Start(ctx, service)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit
		service.Deps.Logger.WithFields(map[string]interface{}{
			"signal": s.String(),
		}).Info("caught signal")

		cancel()

		ctx, cancel = context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		err := server.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}
		service.Deps.Logger.Info("completing background tasks")
		service.Wg.Wait()
		shutdownError <- nil
	}()

	service.Deps.Logger.WithFields(map[string]interface{}{
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

	service.Deps.Logger.WithFields(map[string]interface{}{
		"addr": server.Addr,
	}).Info("stopped server")

	return nil
}
