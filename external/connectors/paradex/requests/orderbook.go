package requests

import (
	"context"
	"fmt"

	"github.com/trishtzy/go-paradex/client/markets"
	"github.com/trishtzy/go-paradex/models"
)

func (s *Service) GetOrderBook(ctx context.Context, market string, depth *int64) (*models.ResponsesAskBidArray, error) {
	params := markets.NewGetOrderbookParams().WithContext(ctx)
	params.SetMarket(market)
	params.SetDepth(depth)
	resp, err := s.client.API().Markets.GetOrderbook(params)
	if err != nil {
		return nil, fmt.Errorf("failed to get order book: %w", err)
	}
	return resp.Payload, nil
}
