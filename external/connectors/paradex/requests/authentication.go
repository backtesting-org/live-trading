package requests

import (
	"context"

	"go.uber.org/zap"
)

func (s *Service) Authenticate(ctx context.Context) error {
	if err := s.client.Authenticate(ctx); err != nil {
		s.logger.Error("Paradex authentication failed", zap.Error(err))
		return err
	}
	s.logger.Info("Paradex authentication successful")
	return nil
}
