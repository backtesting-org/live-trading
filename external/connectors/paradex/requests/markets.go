package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

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

// KlineData represents a single kline/candlestick
type KlineData struct {
	Timestamp int64   // Unix timestamp in milliseconds
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// GetKlines fetches historical klines from Paradex REST API
// Resolution is in minutes: 1, 3, 5, 15, 30, 60
func (s *Service) GetKlines(ctx context.Context, symbol string, resolution int, startAt, endAt int64) ([]KlineData, error) {
	// Use the test net API URL directly
	apiURL := "https://api.testnet.paradex.trade/v1/markets/klines"

	// Build query params
	params := url.Values{}
	params.Add("symbol", symbol)
	params.Add("resolution", strconv.Itoa(resolution))
	params.Add("start_at", strconv.FormatInt(startAt, 10))
	params.Add("end_at", strconv.FormatInt(endAt, 10))
	// CRITICAL: Use mark price for klines since testnet has no trades
	params.Add("price_kind", "mark")

	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResponse struct {
		Results [][]interface{} `json:"results"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to KlineData
	klines := make([]KlineData, 0, len(apiResponse.Results))
	for _, item := range apiResponse.Results {
		if len(item) != 6 {
			s.logger.Warn("Invalid kline data format", "length", len(item))
			continue
		}

		// Parse each field - they come as interface{} (numbers)
		timestamp, ok1 := item[0].(float64)
		open, ok2 := item[1].(float64)
		high, ok3 := item[2].(float64)
		low, ok4 := item[3].(float64)
		close, ok5 := item[4].(float64)
		volume, ok6 := item[5].(float64)

		if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 || !ok6 {
			s.logger.Warn("Failed to parse kline fields", "item", item)
			continue
		}

		klines = append(klines, KlineData{
			Timestamp: int64(timestamp),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		})
	}

	return klines, nil
}
