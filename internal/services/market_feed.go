package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/data/stores/market"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"go.uber.org/zap"
	"github.com/backtesting-org/kronos-sdk/pkg/types/temporal"
)

// MarketDataFeed streams live market data from exchanges to the Kronos store
type MarketDataFeed struct {
	connector connector.Connector // Exchange-agnostic connector
	wsConn    connector.WebSocketConnector // Optional WebSocket connector
	store     market.MarketData
	logger    *zap.Logger
	timeProvider temporal.TimeProvider

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
	conn connector.Connector,
	store market.MarketData,
	logger *zap.Logger,
	exchangeName connector.ExchangeName,
	timeProvider temporal.TimeProvider,
) *MarketDataFeed {
	ctx, cancel := context.WithCancel(context.Background())

	// Check if connector also implements WebSocketConnector
	wsConn, _ := conn.(connector.WebSocketConnector)

	return &MarketDataFeed{
		connector: conn,
		wsConn:    wsConn,
		store:     store,
		logger:    logger,
		timeProvider: timeProvider,
		ctx:       ctx,
		cancel:    cancel,
		assets:    []portfolio.Asset{}, // Will be set when strategies start
		exchanges: []connector.ExchangeName{exchangeName},
	}
}

// Start begins streaming market data
func (mdf *MarketDataFeed) Start() error {
	mdf.logger.Info("Starting market data feed...")

	// Check if connector is available
	if mdf.connector == nil {
		mdf.logger.Warn("Connector not initialized - market data feed disabled")
		return fmt.Errorf("connector not available")
	}

	// Start WebSocket connection if supported
	if mdf.wsConn != nil {
		if err := mdf.wsConn.StartWebSocket(mdf.ctx); err != nil {
			return err
		}

		// Wait briefly for WS to be connected before subscriptions begin
		connectedWaitUntil := mdf.timeProvider.Now().Add(5 * time.Second)
		for !mdf.wsConn.IsWebSocketConnected() && mdf.timeProvider.Now().Before(connectedWaitUntil) {
			mdf.timeProvider.Sleep(100 * time.Millisecond)
		}
		if mdf.wsConn.IsWebSocketConnected() {
			mdf.logger.Info("Websocket connected")
		} else {
			mdf.logger.Warn("Websocket not yet connected; subscriptions will retry when connected")
		}

		// Start consumers for WS-driven updates (orderbooks, klines from trades)
		mdf.wg.Add(3)
		go mdf.consumeOrderbookUpdates()
		go mdf.consumeKlineUpdates()
		go mdf.consumeErrorUpdates()
	}

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

    // Stop WebSocket if supported
	if mdf.wsConn != nil {
		if err := mdf.wsConn.StopWebSocket(); err != nil {
			mdf.logger.Error("Error stopping WebSocket", zap.Error(err))
		}
	}

	mdf.wg.Wait()
	mdf.logger.Info("Market data feed stopped")
}

// updateMarketDataLoop periodically fetches market data and updates the store
func (mdf *MarketDataFeed) updateMarketDataLoop() {
	defer mdf.wg.Done()

	ticker := mdf.timeProvider.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-mdf.ctx.Done():
			return
		case <-ticker.C():
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

	// Get the symbol format for this exchange (e.g., BTC-USD-PERP for perpetual exchanges)
	symbol := mdf.getExchangeSymbol(asset)

	for _, interval := range intervals {
		// Fetch latest 20 klines to ensure overlap with what strategies are reading
		klines, err := mdf.connector.FetchKlines(symbol, interval, 20)
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
	orderBook, err := mdf.connector.FetchOrderBook(asset, connector.TypePerpetual, 50)
	if err != nil {
		return err
	}

	mdf.store.UpdateOrderBook(asset, exchange, connector.TypePerpetual, *orderBook)
	return nil
}

// updateFundingRates fetches and stores funding rate data
func (mdf *MarketDataFeed) updateFundingRates(asset portfolio.Asset, exchange connector.ExchangeName) error {
	fundingRate, err := mdf.connector.FetchFundingRate(asset)
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
    if mdf.connector != nil {
        go mdf.prePopulateKlines(asset)
    }

    // Subscribe to trades (for klines) and orderbook via WS if supported
    if mdf.wsConn != nil {
        if err := mdf.subscribeForAsset(asset); err != nil {
            // If WS not connected, retry in background until success or context canceled
            mdf.logger.Warn("Deferring subscriptions until websocket connects", zap.String("asset", asset.Symbol()), zap.Error(err))
            go mdf.retrySubscribeWhenConnected(asset)
        }
    }
}

func (mdf *MarketDataFeed) subscribeForAsset(asset portfolio.Asset) error {
    if !mdf.wsConn.IsWebSocketConnected() {
        return fmt.Errorf("websocket not connected")
    }
    if err := mdf.wsConn.SubscribeTrades(asset, connector.TypePerpetual); err != nil {
        return err
    }
    if err := mdf.wsConn.SubscribeOrderBook(asset, connector.TypePerpetual); err != nil {
        return err
    }
    // SubscribeKlines ensures trade sub exists; included for clarity
    _ = mdf.wsConn.SubscribeKlines(asset, "5m")
    return nil
}

func (mdf *MarketDataFeed) retrySubscribeWhenConnected(asset portfolio.Asset) {
    ticker := mdf.timeProvider.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()
    for {
        select {
        case <-mdf.ctx.Done():
            return
        case <-ticker.C():
            if mdf.wsConn.IsWebSocketConnected() {
                if err := mdf.subscribeForAsset(asset); err != nil {
                    mdf.logger.Warn("Subscription retry failed", zap.String("asset", asset.Symbol()), zap.Error(err))
                    continue
                }
                mdf.logger.Info("Subscribed to exchange streams for asset", zap.String("asset", asset.Symbol()))
                return
            }
        }
    }
}

// consumeOrderbookUpdates streams WS orderbooks into the store
func (mdf *MarketDataFeed) consumeOrderbookUpdates() {
    defer mdf.wg.Done()
    ch := mdf.wsConn.OrderBookUpdates()
    lastLog := mdf.timeProvider.Now()
    for {
        select {
        case <-mdf.ctx.Done():
            return
        case ob, ok := <-ch:
            if !ok {
                return
            }
            // Use the configured exchange name
            mdf.store.UpdateOrderBook(ob.Asset, mdf.exchanges[0], connector.TypePerpetual, ob)
            mdf.obUpdates++
            if mdf.timeProvider.Since(lastLog) > 30*time.Second {
                mdf.logger.Info("Orderbook stream healthy",
                    zap.Int64("updates_total", mdf.obUpdates),
                    zap.String("last_asset", ob.Asset.Symbol()))
                lastLog = mdf.timeProvider.Now()
            }
        }
    }
}

// consumeKlineUpdates streams WS-built klines into the store
func (mdf *MarketDataFeed) consumeKlineUpdates() {
    defer mdf.wg.Done()
    ch := mdf.wsConn.KlineUpdates()
    lastLog := mdf.timeProvider.Now()
    for {
        select {
        case <-mdf.ctx.Done():
            return
        case k, ok := <-ch:
            if !ok {
                return
            }
            mdf.store.UpdateKline(portfolio.NewAsset(k.Symbol), mdf.exchanges[0], k)
            mdf.klineUpdates++
            if mdf.timeProvider.Since(lastLog) > 30*time.Second {
                mdf.logger.Info("Kline stream healthy",
                    zap.Int64("updates_total", mdf.klineUpdates),
                    zap.String("last_symbol", k.Symbol),
                    zap.String("interval", k.Interval))
                lastLog = mdf.timeProvider.Now()
            }
        }
    }
}

// consumeErrorUpdates logs websocket errors from exchange service
func (mdf *MarketDataFeed) consumeErrorUpdates() {
    defer mdf.wg.Done()
    ch := mdf.wsConn.ErrorChannel()
    for {
        select {
        case <-mdf.ctx.Done():
            return
        case err, ok := <-ch:
            if !ok {
                return
            }
            mdf.logger.Error("Exchange websocket error", zap.Error(err))
        }
    }
}

// prePopulateKlines fetches historical klines for common intervals to pre-populate the store
func (mdf *MarketDataFeed) prePopulateKlines(asset portfolio.Asset) {
	// Common kline intervals used by strategies
	intervals := []string{"1m", "5m", "15m", "1h"}

	// Fetch enough klines for typical strategy needs
	limit := 100

	symbol := mdf.getExchangeSymbol(asset)

	for _, interval := range intervals {
		klines, err := mdf.connector.FetchKlines(symbol, interval, limit)
		if err != nil {
			mdf.logger.Error("Failed to fetch historical klines",
				zap.String("asset", asset.Symbol()),
				zap.String("interval", interval),
				zap.Error(err))
			continue
		}

		// Add klines to the store
		for _, kline := range klines {
			mdf.store.UpdateKline(asset, mdf.exchanges[0], kline)
		}
	}
}

// getExchangeSymbol gets the exchange-specific symbol format for an asset
// This method handles exchange-specific symbol formatting
func (mdf *MarketDataFeed) getExchangeSymbol(asset portfolio.Asset) string {
	// For exchanges that support the GetPerpSymbol method, use it
	if symbolGetter, ok := mdf.connector.(interface{ GetPerpSymbol(portfolio.Asset) string }); ok {
		return symbolGetter.GetPerpSymbol(asset)
	}

	// Default: use the asset symbol as-is
	return asset.Symbol()
}
