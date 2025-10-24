package requests

import (
	"context"
	"fmt"

	"github.com/trishtzy/go-paradex/client/markets"
	"github.com/trishtzy/go-paradex/models"
)

func (s *Service) GetMarkets(ctx context.Context) ([]*models.ResponsesMarketResp, error) {
	params := markets.NewGetMarketsParams().WithContext(ctx)
	resp, err := s.client.API().Markets.GetMarkets(params)
	if err != nil {
		return nil, fmt.Errorf("failed to get markets: %w", err)
	}
	return resp.Payload.Results, nil
}

func (s *Service) GetPrice(ctx context.Context, market string) (*models.ResponsesBBOResp, error) {
	params := markets.NewGetBboParams().WithContext(ctx)
	params.SetMarket(market)
	resp, err := s.client.API().Markets.GetBbo(params)
	if err != nil {
		return nil, fmt.Errorf("failed to get market price: %w", err)
	}
	if resp.Payload == nil {
		return nil, fmt.Errorf("market not found: %s", market)
	}
	return resp.Payload, nil
}
