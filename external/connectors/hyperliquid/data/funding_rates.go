package data

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sonirico/go-hyperliquid"
)

func (m *MarketDataService) GetCurrentFundingRatesMap() (map[string]map[string]float64, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	payload := `{"type": "metaAndAssetCtxs"}`
	resp, err := client.Post("https://api.hyperliquid.xyz/info",
		"application/json", strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse as array with two elements: [universeObject, assetCtxsArray]
	var responseArray []interface{}
	if err := json.Unmarshal(body, &responseArray); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response array: %w", err)
	}

	if len(responseArray) < 2 {
		return nil, fmt.Errorf("unexpected response format: expected 2 elements")
	}

	// First element contains universe
	universeObj := responseArray[0].(map[string]interface{})
	universe := universeObj["universe"].([]interface{})

	// Second element is assetCtxs array
	assetCtxs := responseArray[1].([]interface{})

	cleanData := make(map[string]map[string]float64)

	// Match universe with assetCtxs by index
	for i := 0; i < len(universe) && i < len(assetCtxs); i++ {
		universeItem := universe[i].(map[string]interface{})
		assetCtx := assetCtxs[i].(map[string]interface{})

		assetName := universeItem["name"].(string)

		cleanData[assetName] = map[string]float64{
			"funding":      parseFloat(assetCtx["funding"]),
			"markPrice":    parseFloat(assetCtx["markPx"]),
			"openInterest": parseFloat(assetCtx["openInterest"]),
			"premium":      parseFloat(assetCtx["premium"]),
			"dayVolume":    parseFloat(assetCtx["dayNtlVlm"]),
			"midPrice":     parseFloat(assetCtx["midPx"]),
			"oraclePrice":  parseFloat(assetCtx["oraclePx"]),
		}
	}

	return cleanData, nil
}

func (m *MarketDataService) GetHistoricalFundingRates(coin string, startTime, endTime int64) ([]hyperliquid.FundingHistory, error) {
	// Convert both timestamps from seconds to milliseconds
	startTimeMs := startTime * millisecondsPerSecond
	endTimeMs := endTime * millisecondsPerSecond

	return m.info.FundingHistory(coin, startTimeMs, &endTimeMs)
}

func parseFloat(val interface{}) float64 {
	if str, ok := val.(string); ok {
		if f, err := strconv.ParseFloat(str, 64); err == nil {
			return f
		}
	}
	return 0.0
}
