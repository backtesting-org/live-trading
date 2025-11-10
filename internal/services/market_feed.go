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

    // instrumentation
    obUpdates   int64
    klineUpdates int64
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

    // Wait briefly for WS to be connected before subscriptions begin
    connectedWaitUntil := time.Now().Add(5 * time.Second)
    for !mdf.paradexConnector.IsWebSocketConnected() && time.Now().Before(connectedWaitUntil) {
        time.Sleep(100 * time.Millisecond)
    }
    if mdf.paradexConnector.IsWebSocketConnected() {
        mdf.logger.Info("Paradex websocket connected")
    } else {
        mdf.logger.Warn("Paradex websocket not yet connected; subscriptions will retry when connected")
    }

    // Start consumers for WS-driven updates (orderbooks, klines from trades)
    mdf.wg.Add(3)
    go mdf.consumeOrderbookUpdates()
    go mdf.consumeKlineUpdates()
    go mdf.consumeErrorUpdates()

    // Start periodic market data updates (REST fallbacks)
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
	// Fetch recent klines for common intervals to keep data fresh
	// This is especially important when there are no trades (testnet scenario)
	intervals := []string{"1m", "5m", "15m", "1h"}
	symbol := mdf.paradexConnector.GetPerpSymbol(asset)

	for _, interval := range intervals {
		// Fetch latest 20 klines to ensure overlap with what strategies are reading
		klines, err := mdf.paradexConnector.FetchKlines(symbol, interval, 20)
		if err != nil {
			mdf.logger.Warn("Failed to fetch klines for interval",
				zap.String("asset", asset.Symbol()),
				zap.String("interval", interval),
				zap.Error(err))
			continue
		}

		// Update the store with fresh klines
		for _, kline := range klines {
			mdf.store.UpdateKline(asset, exchange, kline)
		}
	}

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

    // Pre-populate historical klines for common intervals
    if mdf.paradexConnector != nil {
        go mdf.prePopulateKlines(asset)
    }

    // Subscribe to trades (for klines) and orderbook via WS
    if mdf.paradexConnector != nil {
        if err := mdf.subscribeForAsset(asset); err != nil {
            // If WS not connected, retry in background until success or context canceled
            mdf.logger.Warn("Deferring subscriptions until websocket connects", zap.String("asset", asset.Symbol()), zap.Error(err))
            go mdf.retrySubscribeWhenConnected(asset)
        }
    }
}

func (mdf *MarketDataFeed) subscribeForAsset(asset portfolio.Asset) error {
    if !mdf.paradexConnector.IsWebSocketConnected() {
        return fmt.Errorf("websocket not connected")
    }
    if err := mdf.paradexConnector.SubscribeTrades(asset, connector.TypePerpetual); err != nil {
        return err
    }
    if err := mdf.paradexConnector.SubscribeOrderBook(asset, connector.TypePerpetual); err != nil {
        return err
    }
    // SubscribeKlines ensures trade sub exists; included for clarity
    _ = mdf.paradexConnector.SubscribeKlines(asset, "5m")
    return nil
}

func (mdf *MarketDataFeed) retrySubscribeWhenConnected(asset portfolio.Asset) {
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()
    for {
        select {
        case <-mdf.ctx.Done():
            return
        case <-ticker.C:
            if mdf.paradexConnector.IsWebSocketConnected() {
                if err := mdf.subscribeForAsset(asset); err != nil {
                    mdf.logger.Warn("Subscription retry failed", zap.String("asset", asset.Symbol()), zap.Error(err))
                    continue
                }
                mdf.logger.Info("Subscribed to Paradex streams for asset", zap.String("asset", asset.Symbol()))
                return
            }
        }
    }
}

// consumeOrderbookUpdates streams WS orderbooks into the store
func (mdf *MarketDataFeed) consumeOrderbookUpdates() {
    defer mdf.wg.Done()
    ch := mdf.paradexConnector.OrderBookUpdates()
    lastLog := time.Now()
    for {
        select {
        case <-mdf.ctx.Done():
            return
        case ob, ok := <-ch:
            if !ok {
                return
            }
            // Use Paradex exchange name
            mdf.store.UpdateOrderBook(ob.Asset, connector.Paradex, connector.TypePerpetual, ob)
            mdf.obUpdates++
            if time.Since(lastLog) > 30*time.Second {
                mdf.logger.Info("Orderbook stream healthy",
                    zap.Int64("updates_total", mdf.obUpdates),
                    zap.String("last_asset", ob.Asset.Symbol()))
                lastLog = time.Now()
            }
        }
    }
}

// consumeKlineUpdates streams WS-built klines into the store
func (mdf *MarketDataFeed) consumeKlineUpdates() {
    defer mdf.wg.Done()
    ch := mdf.paradexConnector.KlineUpdates()
    lastLog := time.Now()
    for {
        select {
        case <-mdf.ctx.Done():
            return
        case k, ok := <-ch:
            if !ok {
                return
            }
            mdf.store.UpdateKline(portfolio.NewAsset(k.Symbol), connector.Paradex, k)
            mdf.klineUpdates++
            if time.Since(lastLog) > 30*time.Second {
                mdf.logger.Info("Kline stream healthy",
                    zap.Int64("updates_total", mdf.klineUpdates),
                    zap.String("last_symbol", k.Symbol),
                    zap.String("interval", k.Interval))
                lastLog = time.Now()
            }
        }
    }
}

// consumeErrorUpdates logs websocket errors from Paradex service
func (mdf *MarketDataFeed) consumeErrorUpdates() {
    defer mdf.wg.Done()
    ch := mdf.paradexConnector.ErrorChannel()
    for {
        select {
        case <-mdf.ctx.Done():
            return
        case err, ok := <-ch:
            if !ok {
                return
            }
            mdf.logger.Error("Paradex websocket error", zap.Error(err))
        }
    }
}

// prePopulateKlines fetches historical klines for common intervals to pre-populate the store
func (mdf *MarketDataFeed) prePopulateKlines(asset portfolio.Asset) {
	// Common kline intervals used by strategies
	intervals := []string{"1m", "5m", "15m", "1h"}

	// Fetch enough klines for typical strategy needs
	limit := 100

	symbol := mdf.paradexConnector.GetPerpSymbol(asset)

	for _, interval := range intervals {
		klines, err := mdf.paradexConnector.FetchKlines(symbol, interval, limit)
		if err != nil {
			mdf.logger.Error("Failed to fetch historical klines",
				zap.String("asset", asset.Symbol()),
				zap.String("interval", interval),
				zap.Error(err))
			continue
		}

		// Add klines to the store
		for _, kline := range klines {
			mdf.store.UpdateKline(asset, connector.Paradex, kline)
		}
	}
}
