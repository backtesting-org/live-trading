package data

import (
	"fmt"

	"github.com/sonirico/go-hyperliquid"
)

// AssetContext represents the parsed asset context data
type AssetContext struct {
	Name         string
	Funding      string
	MarkPrice    string
	OraclePrice  string
	Premium      string
	OpenInterest string
}

// GetAssetContext returns the asset context for a specific coin
func (m *marketDataService) GetAssetContext(coin string) (*AssetContext, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}

	metaAndCtx, err := info.MetaAndAssetCtxs()
	if err != nil {
		return nil, fmt.Errorf("failed to get asset contexts: %w", err)
	}

	if len(metaAndCtx) < 2 {
		return nil, fmt.Errorf("invalid response format")
	}

	universeData, ok := metaAndCtx["universe"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid universe data")
	}

	assetCtxs, ok := metaAndCtx["assetCtxs"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid asset contexts")
	}

	for i := 0; i < len(universeData) && i < len(assetCtxs); i++ {
		assetInfo, ok := universeData[i].(map[string]interface{})
		if !ok {
			continue
		}

		name, ok := assetInfo["name"].(string)
		if !ok || name != coin {
			continue
		}

		ctx, ok := assetCtxs[i].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid context for asset %s", coin)
		}

		fundingStr, okFunding := ctx["funding"].(string)
		markPxStr, okMark := ctx["markPx"].(string)
		oraclePxStr, okOracle := ctx["oraclePx"].(string)
		premiumStr, okPremium := ctx["premium"].(string)
		openInterestStr, okOI := ctx["openInterest"].(string)

		if !okFunding || !okMark || !okOracle || !okPremium || !okOI {
			return nil, fmt.Errorf("missing required fields for asset %s (funding=%v, mark=%v, oracle=%v, premium=%v, oi=%v)",
				coin, okFunding, okMark, okOracle, okPremium, okOI)
		}

		return &AssetContext{
			Name:         name,
			Funding:      fundingStr,
			MarkPrice:    markPxStr,
			OraclePrice:  oraclePxStr,
			Premium:      premiumStr,
			OpenInterest: openInterestStr,
		}, nil
	}

	return nil, fmt.Errorf("asset %s not found", coin)
}

// GetAllAssetContexts returns asset contexts for all assets
func (m *marketDataService) GetAllAssetContexts() ([]AssetContext, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}

	metaAndCtx, err := info.MetaAndAssetCtxs()
	if err != nil {
		return nil, fmt.Errorf("failed to get asset contexts: %w", err)
	}

	if len(metaAndCtx) < 2 {
		return nil, fmt.Errorf("invalid response format")
	}

	universeData, ok := metaAndCtx["universe"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid universe data")
	}

	assetCtxs, ok := metaAndCtx["assetCtxs"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid asset contexts")
	}

	var contexts []AssetContext

	for i := 0; i < len(universeData) && i < len(assetCtxs); i++ {
		assetInfo, ok := universeData[i].(map[string]interface{})
		if !ok {
			continue
		}

		name, ok := assetInfo["name"].(string)
		if !ok {
			continue
		}

		ctx, ok := assetCtxs[i].(map[string]interface{})
		if !ok {
			continue
		}

		fundingStr, okFunding := ctx["funding"].(string)
		markPxStr, okMark := ctx["markPx"].(string)
		oraclePxStr, okOracle := ctx["oraclePx"].(string)
		premiumStr, okPremium := ctx["premium"].(string)
		openInterestStr, okOI := ctx["openInterest"].(string)

		if !okFunding || !okMark || !okOracle || !okPremium || !okOI {
			// Skip assets with incomplete data rather than failing
			continue
		}

		contexts = append(contexts, AssetContext{
			Name:         name,
			Funding:      fundingStr,
			MarkPrice:    markPxStr,
			OraclePrice:  oraclePxStr,
			Premium:      premiumStr,
			OpenInterest: openInterestStr,
		})
	}

	return contexts, nil
}

// GetHistoricalFundingRates returns historical funding rates for a specific coin
// This is a public endpoint and doesn't require authentication
func (m *marketDataService) GetHistoricalFundingRates(coin string, startTime, endTime int64) ([]hyperliquid.FundingHistory, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}

	// Convert both timestamps from seconds to milliseconds
	startTimeMs := startTime * millisecondsPerSecond
	endTimeMs := endTime * millisecondsPerSecond

	return info.FundingHistory(coin, startTimeMs, &endTimeMs)
}
