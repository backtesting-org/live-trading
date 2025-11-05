package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio/store"
	"github.com/backtesting-org/live-trading/internal/exchanges/paradex"
	"go.uber.org/zap"
)

// MarketDataFeed streams live market data from exchanges to the Kronos store
type MarketDataFeed struct {
	paradexConnector *paradex.Paradex
	store            store.Store
	logger           *zap.Logger

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Assets to monitor
	assets    []portfolio.Asset
	exchanges []connector.ExchangeName
}

// NewMarketDataFeed creates a new market data feed service
func NewMarketDataFeed(
	paradexConnector *paradex.Paradex,
	store store.Store,
	logger *zap.Logger,
) *MarketDataFeed {
	ctx, cancel := context.WithCancel(context.Background())

	return &MarketDataFeed{
		paradexConnector: paradexConnector,
		store:            store,
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
		assets:           []portfolio.Asset{}, // Will be set when strategies start
		exchanges:        []connector.ExchangeName{connector.Paradex},
	}
}

// Start begins streaming market data
func (mdf *MarketDataFeed) Start() error {
	mdf.logger.Info("Starting market data feed...")

	// Check if connector is available
	if mdf.paradexConnector == nil {
		mdf.logger.Warn("Paradex connector not initialized - market data feed disabled")
		return fmt.Errorf("paradex connector not available")
	}

	// Start WebSocket connection
	if err := mdf.paradexConnector.StartWebSocket(mdf.ctx); err != nil {
		return err
	}

	// Start periodic market data updates
	// Assets will be added dynamically when strategies start
	mdf.wg.Add(1)
	go mdf.updateMarketDataLoop()

	mdf.logger.Info("Market data feed started successfully")
	return nil
}

// Stop gracefully shuts down the market data feed
func (mdf *MarketDataFeed) Stop() {
	mdf.logger.Info("Stopping market data feed...")
	mdf.cancel()

	// Stop WebSocket
	if err := mdf.paradexConnector.StopWebSocket(); err != nil {
		mdf.logger.Error("Error stopping WebSocket", zap.Error(err))
	}

	mdf.wg.Wait()
	mdf.logger.Info("Market data feed stopped")
}

// updateMarketDataLoop periodically fetches market data and updates the store
func (mdf *MarketDataFeed) updateMarketDataLoop() {
	defer mdf.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-mdf.ctx.Done():
			return
		case <-ticker.C:
			mdf.updateMarketData()
		}
	}
}

// updateMarketData fetches fresh market data for all monitored assets
func (mdf *MarketDataFeed) updateMarketData() {
	for _, asset := range mdf.assets {
		for _, exchange := range mdf.exchanges {
			// Update klines
			if err := mdf.updateKlines(asset, exchange); err != nil {
				mdf.logger.Error("Failed to update klines",
					zap.String("asset", asset.Symbol()),
					zap.String("exchange", string(exchange)),
					zap.Error(err))
			}

			// Update orderbook
			if err := mdf.updateOrderBook(asset, exchange); err != nil {
				mdf.logger.Error("Failed to update orderbook",
					zap.String("asset", asset.Symbol()),
					zap.String("exchange", string(exchange)),
					zap.Error(err))
			}

			// Update funding rates
			if err := mdf.updateFundingRates(asset, exchange); err != nil {
				mdf.logger.Error("Failed to update funding rates",
					zap.String("asset", asset.Symbol()),
					zap.String("exchange", string(exchange)),
					zap.Error(err))
			}
		}
	}
}

// updateKlines fetches and stores recent klines
func (mdf *MarketDataFeed) updateKlines(asset portfolio.Asset, exchange connector.ExchangeName) error {
	// Klines not implemented in Paradex yet - skip for now
	// intervals := []string{"1m", "5m", "15m", "1h"}
	// for _, interval := range intervals {
	// 	klines, err := mdf.paradexConnector.FetchKlines(asset.Symbol(), interval, 100)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	for _, kline := range klines {
	// 		mdf.store.UpdateKline(asset, exchange, kline)
	// 	}
	// }
	return nil
}

// updateOrderBook fetches and stores current orderbook
func (mdf *MarketDataFeed) updateOrderBook(asset portfolio.Asset, exchange connector.ExchangeName) error {
	orderBook, err := mdf.paradexConnector.FetchOrderBook(asset, connector.TypePerpetual, 50)
	if err != nil {
		return err
	}

	mdf.store.UpdateOrderBook(asset, exchange, connector.TypePerpetual, *orderBook)
	return nil
}

// updateFundingRates fetches and stores funding rate data
func (mdf *MarketDataFeed) updateFundingRates(asset portfolio.Asset, exchange connector.ExchangeName) error {
	fundingRate, err := mdf.paradexConnector.FetchFundingRate(asset)
	if err != nil {
		return err
	}

	mdf.store.UpdateFundingRate(asset, exchange, *fundingRate)
	return nil
}

// AddAsset adds an asset to monitor (called when strategy starts)
func (mdf *MarketDataFeed) AddAsset(asset portfolio.Asset) {
	// Check if already monitoring
	for _, a := range mdf.assets {
		if a.Symbol() == asset.Symbol() {
			return
		}
	}

	mdf.assets = append(mdf.assets, asset)
	mdf.logger.Info("Added asset to market feed", zap.String("asset", asset.Symbol()))
}
