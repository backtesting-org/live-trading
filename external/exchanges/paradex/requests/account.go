package requests

import (
	"context"
	"fmt"

	"github.com/trishtzy/go-paradex/client/account"
	"github.com/trishtzy/go-paradex/models"
)

func (s *Service) GetAccount(ctx context.Context) (*models.ResponsesAccountSummaryResponse, error) {
	params := account.NewGetAccountParams().WithContext(ctx)
	resp, err := s.client.API().Account.GetAccount(params, s.client.AuthWriter(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}

	return resp.Payload, nil
}

func (s *Service) GetAssetBalances(ctx context.Context) ([]*models.ResponsesBalanceResp, error) {
	params := account.NewGetBalanceParams().WithContext(ctx)
	resp, err := s.client.API().Account.GetBalance(params, s.client.AuthWriter(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get account balance: %w", err)
	}
	return resp.Payload.Results, nil
}

func (s *Service) GetProfile(ctx context.Context) (*models.ResponsesAccountProfileResp, error) {
	params := account.NewGetAccountProfileParams().WithContext(ctx)
	resp, err := s.client.API().Account.GetAccountProfile(params, s.client.AuthWriter(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	return resp.Payload, nil
}

func (s *Service) GetTradeHistory(ctx context.Context, market *string, limit *int64) (*models.ResponsesGetFillsResp, error) {
	params := account.NewGetFillsParams().WithContext(ctx)
	params.SetMarket(market)
	if limit != nil {
		params.SetPageSize(limit)
	}
	resp, err := s.client.API().Account.GetFills(params, s.client.AuthWriter(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get trade history: %w", err)
	}
	return resp.Payload, nil
}

func (s *Service) GetUserPositions(ctx context.Context) (*models.ResponsesGetPositionsResp, error) {
	params := account.NewGetPositionsParams().WithContext(ctx)
	resp, err := s.client.API().Account.GetPositions(params, s.client.AuthWriter(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get user orders: %w", err)
	}
	return resp.Payload, nil
}
