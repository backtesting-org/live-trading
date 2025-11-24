package hyperliquid

import (
	"fmt"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/shopspring/decimal"
)

func (h *Hyperliquid) FetchCurrentFundingRates() (map[portfolio.Asset]connector.FundingRate, error) {
	rawData, err := h.marketData.GetCurrentFundingRatesMap()
	if err != nil {
		return nil, err
	}

	fundingRates := make(map[portfolio.Asset]connector.FundingRate)

	for symbol, data := range rawData {
		asset := portfolio.NewAsset(symbol)

		fundingRates[asset] = connector.FundingRate{
			CurrentRate:     decimal.NewFromFloat(data["funding"]),
			Timestamp:       time.Now(),
			MarkPrice:       decimal.NewFromFloat(data["markPrice"]),
			IndexPrice:      decimal.NewFromFloat(data["oraclePrice"]),
			Premium:         decimal.NewFromFloat(data["premium"]),
			NextFundingTime: time.Now().Add(time.Hour),
		}
	}

	return fundingRates, nil
}

func (h *Hyperliquid) FetchFundingRate(asset portfolio.Asset) (*connector.FundingRate, error) {
	allRates, err := h.FetchCurrentFundingRates()
	if err != nil {
		return nil, err
	}

	rate, exists := allRates[asset]
	if !exists {
		return nil, fmt.Errorf("funding rate not found for symbol: %s", asset.Symbol())
	}

	return &rate, nil
}

func (h *Hyperliquid) FetchHistoricalFundingRates(symbol portfolio.Asset, startTime, endTime int64) ([]connector.HistoricalFundingRate, error) {
	rawData, err := h.marketData.GetHistoricalFundingRates(symbol.Symbol(), startTime, endTime)
	if err != nil {
		return nil, err
	}

	var rates []connector.HistoricalFundingRate
	for _, entry := range rawData {
		fundingRate, err := decimal.NewFromString(entry.FundingRate)

		if err != nil {
			return nil, fmt.Errorf("invalid funding rate %s for symbol %s: %w", entry.FundingRate, symbol.Symbol(), err)
		}

		rates = append(rates, connector.HistoricalFundingRate{
			FundingRate: fundingRate,
			Timestamp:   time.Unix(entry.Time/1000, 0),
		})
	}

	return rates, nil
}

func (h *Hyperliquid) SupportsFundingRates() bool {
	return true
}
