package requests

import (
	"context"
	"fmt"

	"github.com/trishtzy/go-paradex/client/system"
	"github.com/trishtzy/go-paradex/models"
)

func (s *Service) GetSystemConfig(ctx context.Context) (*models.ResponsesSystemConfigResponse, error) {
	params := system.NewGetSystemConfigParams().WithContext(ctx)
	resp, err := s.client.API().System.GetSystemConfig(params)
	if err != nil {
		return nil, fmt.Errorf("failed to get system config: %w", err)
	}
	return resp.Payload, nil
}

func (s *Service) GetSystemState(ctx context.Context) (*models.ResponsesSystemStateResponse, error) {
	params := system.NewGetSystemStateParams().WithContext(ctx)
	resp, err := s.client.API().System.GetSystemState(params)
	if err != nil {
		return nil, fmt.Errorf("failed to get system state: %w", err)
	}
	return resp.Payload, nil
}

func (s *Service) GetSystemTime(ctx context.Context) (*models.ResponsesSystemTimeResponse, error) {
	params := system.NewGetSystemTimeParams().WithContext(ctx)
	resp, err := s.client.API().System.GetSystemTime(params)
	if err != nil {
		return nil, fmt.Errorf("failed to get system time: %w", err)
	}
	return resp.Payload, nil
}

func (s *Service) CheckHealth(ctx context.Context) error {
	state, err := s.GetSystemState(ctx)
	if err != nil {
		return err
	}
	if state.Status.ResponsesSystemStatus != models.ResponsesSystemStatusOk {
		return fmt.Errorf("system not healthy: %s", state.Status)
	}
	return nil
}
