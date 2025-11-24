package hyperliquid

import (
	"fmt"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/shopspring/decimal"
)

func (h *hyperliquid) FetchCurrentFundingRates() (map[portfolio.Asset]connector.FundingRate, error) {
	contexts, err := h.marketData.GetAllAssetContexts()
	if err != nil {
		return nil, fmt.Errorf("failed to get asset contexts: %w", err)
	}

	fundingRates := make(map[portfolio.Asset]connector.FundingRate)

	for _, ctx := range contexts {
		asset := portfolio.NewAsset(ctx.Name)

		funding, _ := decimal.NewFromString(ctx.Funding)
		markPrice, _ := decimal.NewFromString(ctx.MarkPrice)
		oraclePrice, _ := decimal.NewFromString(ctx.OraclePrice)

		fundingRates[asset] = connector.FundingRate{
			CurrentRate:     funding,
			Timestamp:       time.Now(),
			MarkPrice:       markPrice,
			IndexPrice:      oraclePrice,
			NextFundingTime: time.Now().Add(time.Hour),
		}
	}

	return fundingRates, nil
}

func (h *hyperliquid) FetchFundingRate(asset portfolio.Asset) (*connector.FundingRate, error) {
	ctx, err := h.marketData.GetAssetContext(asset.Symbol())
	if err != nil {
		return nil, fmt.Errorf("failed to get asset context: %w", err)
	}

	funding, _ := decimal.NewFromString(ctx.Funding)
	markPrice, _ := decimal.NewFromString(ctx.MarkPrice)
	oraclePrice, _ := decimal.NewFromString(ctx.OraclePrice)

	return &connector.FundingRate{
		CurrentRate:     funding,
		Timestamp:       time.Now(),
		MarkPrice:       markPrice,
		IndexPrice:      oraclePrice,
		NextFundingTime: time.Now().Add(time.Hour),
	}, nil
}

func (h *hyperliquid) FetchHistoricalFundingRates(symbol portfolio.Asset, startTime, endTime int64) ([]connector.HistoricalFundingRate, error) {
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

func (h *hyperliquid) SupportsFundingRates() bool {
	return true
}
