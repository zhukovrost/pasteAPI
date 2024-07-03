package autoclean

import (
	"context"
	"github.com/zhukovrost/pasteAPI/internal/service"
	"time"
)

func Start(ctx context.Context, s *service.Service) {
	ticker := time.NewTicker(time.Hour * 1)
	defer ticker.Stop()

	clean(ctx, s)

	for {
		select {
		case <-ticker.C:
			clean(ctx, s)
		case <-ctx.Done():
			s.Logger.Info("Stopping auto-cleaning...")
			return
		}
	}
}

func clean(ctx context.Context, s *service.Service) {
	query := `
		DELETE FROM pastes
		WHERE expires_at < NOW();
		
		DELETE FROM tokens
		WHERE expiry < NOW();`

	_, err := s.DB.ExecContext(ctx, query)
	s.Logger.Info("cleaning database...")
	if err != nil {
		s.Logger.Errorf("Error while cleaning pastes and tokens: %v", err)
	}
	s.Logger.Info("database has been cleaned")
}
