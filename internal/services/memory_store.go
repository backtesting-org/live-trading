package services

import (
	"sync"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio/store"
)

// MemoryStore implements the Store interface with in-memory storage
type MemoryStore struct {
	mu                     sync.RWMutex
	fundingRates           map[portfolio.Asset]store.FundingRateMap
	historicalFundingRates map[portfolio.Asset]store.HistoricalFundingMap
	orderBooks             map[portfolio.Asset]store.OrderBookMap
	assetPrices            map[portfolio.Asset]store.PriceMap
	klines                 map[portfolio.Asset]store.KlineMap
	lastUpdated            store.LastUpdatedMap
	notifier               func()
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		fundingRates:           make(map[portfolio.Asset]store.FundingRateMap),
		historicalFundingRates: make(map[portfolio.Asset]store.HistoricalFundingMap),
		orderBooks:             make(map[portfolio.Asset]store.OrderBookMap),
		assetPrices:            make(map[portfolio.Asset]store.PriceMap),
		klines:                 make(map[portfolio.Asset]store.KlineMap),
		lastUpdated:            make(store.LastUpdatedMap),
	}
}

func (ms *MemoryStore) UpdateFundingRate(asset portfolio.Asset, exchangeName connector.ExchangeName, rate connector.FundingRate) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.fundingRates[asset] == nil {
		ms.fundingRates[asset] = make(store.FundingRateMap)
	}
	ms.fundingRates[asset][exchangeName] = rate

	ms.UpdateLastUpdated(store.UpdateKey{
		DataType: store.DataKeyFundingRates,
		Asset:    asset,
		Exchange: exchangeName,
	})
}

func (ms *MemoryStore) UpdateFundingRates(exchangeName connector.ExchangeName, rates map[portfolio.Asset]connector.FundingRate) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for asset, rate := range rates {
		if ms.fundingRates[asset] == nil {
			ms.fundingRates[asset] = make(store.FundingRateMap)
		}
		ms.fundingRates[asset][exchangeName] = rate

		ms.lastUpdated[store.UpdateKey{
			DataType: store.DataKeyFundingRates,
			Asset:    asset,
			Exchange: exchangeName,
		}] = time.Now()
	}

	if ms.notifier != nil {
		ms.notifier()
	}
}

func (ms *MemoryStore) GetFundingRatesForAsset(asset portfolio.Asset) store.FundingRateMap {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.fundingRates[asset]
}

func (ms *MemoryStore) GetFundingRate(asset portfolio.Asset, exchangeName connector.ExchangeName) *connector.FundingRate {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if rates, ok := ms.fundingRates[asset]; ok {
		if rate, exists := rates[exchangeName]; exists {
			return &rate
		}
	}
	return nil
}

func (ms *MemoryStore) GetAllAssetsWithFundingRates() []portfolio.Asset {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	assets := make([]portfolio.Asset, 0, len(ms.fundingRates))
	for asset := range ms.fundingRates {
		assets = append(assets, asset)
	}
	return assets
}

func (ms *MemoryStore) UpdateHistoricalFundingRates(asset portfolio.Asset, exchangeName connector.ExchangeName, rates []connector.HistoricalFundingRate) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.historicalFundingRates[asset] == nil {
		ms.historicalFundingRates[asset] = make(store.HistoricalFundingMap)
	}
	ms.historicalFundingRates[asset][exchangeName] = rates

	ms.lastUpdated[store.UpdateKey{
		DataType: store.DataKeyHistoricalFunding,
		Asset:    asset,
		Exchange: exchangeName,
	}] = time.Now()
}

func (ms *MemoryStore) GetHistoricalFundingRatesForAsset(asset portfolio.Asset) store.HistoricalFundingMap {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.historicalFundingRates[asset]
}

func (ms *MemoryStore) UpdateOrderBook(asset portfolio.Asset, exchangeName connector.ExchangeName, orderBookType connector.Instrument, orderBook connector.OrderBook) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.orderBooks[asset] == nil {
		ms.orderBooks[asset] = make(store.OrderBookMap)
	}
	if ms.orderBooks[asset][exchangeName] == nil {
		ms.orderBooks[asset][exchangeName] = make(map[connector.Instrument]*connector.OrderBook)
	}
	ms.orderBooks[asset][exchangeName][orderBookType] = &orderBook

	ms.UpdateLastUpdated(store.UpdateKey{
		DataType: store.DataKeyOrderBooks,
		Asset:    asset,
		Exchange: exchangeName,
	})
}

func (ms *MemoryStore) GetOrderBooks(asset portfolio.Asset) store.OrderBookMap {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.orderBooks[asset]
}

func (ms *MemoryStore) GetOrderBook(asset portfolio.Asset, exchangeName connector.ExchangeName, orderBookType connector.Instrument) *connector.OrderBook {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if books, ok := ms.orderBooks[asset]; ok {
		if exchangeBooks, exists := books[exchangeName]; exists {
			return exchangeBooks[orderBookType]
		}
	}
	return nil
}

func (ms *MemoryStore) GetAllAssetsWithOrderBooks() []portfolio.Asset {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	assets := make([]portfolio.Asset, 0, len(ms.orderBooks))
	for asset := range ms.orderBooks {
		assets = append(assets, asset)
	}
	return assets
}

func (ms *MemoryStore) UpdateAssetPrice(asset portfolio.Asset, exchangeName connector.ExchangeName, price connector.Price) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.assetPrices[asset] == nil {
		ms.assetPrices[asset] = make(store.PriceMap)
	}
	ms.assetPrices[asset][exchangeName] = price

	ms.UpdateLastUpdated(store.UpdateKey{
		DataType: store.DataKeyAssetPrice,
		Asset:    asset,
		Exchange: exchangeName,
	})
}

func (ms *MemoryStore) UpdateAssetPrices(asset portfolio.Asset, prices map[connector.ExchangeName]connector.Price) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.assetPrices[asset] == nil {
		ms.assetPrices[asset] = make(store.PriceMap)
	}

	for exchange, price := range prices {
		ms.assetPrices[asset][exchange] = price
		ms.lastUpdated[store.UpdateKey{
			DataType: store.DataKeyAssetPrice,
			Asset:    asset,
			Exchange: exchange,
		}] = time.Now()
	}

	if ms.notifier != nil {
		ms.notifier()
	}
}

func (ms *MemoryStore) GetAssetPrice(asset portfolio.Asset, exchangeName connector.ExchangeName) *connector.Price {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if prices, ok := ms.assetPrices[asset]; ok {
		if price, exists := prices[exchangeName]; exists {
			return &price
		}
	}
	return nil
}

func (ms *MemoryStore) GetAssetPrices(asset portfolio.Asset) store.PriceMap {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.assetPrices[asset]
}

func (ms *MemoryStore) UpdateKline(asset portfolio.Asset, exchangeName connector.ExchangeName, kline connector.Kline) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.klines[asset] == nil {
		ms.klines[asset] = make(store.KlineMap)
	}
	if ms.klines[asset][exchangeName] == nil {
		ms.klines[asset][exchangeName] = make(map[string][]connector.Kline)
	}

	interval := kline.Interval
	klines := ms.klines[asset][exchangeName][interval]

	// Add or update kline
	found := false
	for i, k := range klines {
		if k.OpenTime.Equal(kline.OpenTime) {
			klines[i] = kline
			found = true
			break
		}
	}

	if !found {
		klines = append(klines, kline)
	}

	ms.klines[asset][exchangeName][interval] = klines

	ms.UpdateLastUpdated(store.UpdateKey{
		DataType: store.DataKeyKlines,
		Asset:    asset,
		Exchange: exchangeName,
	})
}

func (ms *MemoryStore) GetKlines(asset portfolio.Asset, exchangeName connector.ExchangeName, interval string, limit int) []connector.Kline {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if assetKlines, ok := ms.klines[asset]; ok {
		if exchangeKlines, exists := assetKlines[exchangeName]; exists {
			if klines, found := exchangeKlines[interval]; found {
				if limit > 0 && len(klines) > limit {
					return klines[len(klines)-limit:]
				}
				return klines
			}
		}
	}
	return nil
}

func (ms *MemoryStore) GetKlinesSince(asset portfolio.Asset, exchangeName connector.ExchangeName, interval string, since time.Time) []connector.Kline {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if assetKlines, ok := ms.klines[asset]; ok {
		if exchangeKlines, exists := assetKlines[exchangeName]; exists {
			if klines, found := exchangeKlines[interval]; found {
				var result []connector.Kline
				for _, k := range klines {
					if k.OpenTime.After(since) || k.OpenTime.Equal(since) {
						result = append(result, k)
					}
				}
				return result
			}
		}
	}
	return nil
}

func (ms *MemoryStore) SetOrchestratorNotifier(notifier func()) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.notifier = notifier
}

func (ms *MemoryStore) GetLastUpdated() store.LastUpdatedMap {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.lastUpdated
}

func (ms *MemoryStore) UpdateLastUpdated(key store.UpdateKey) {
	// Must be called with lock held
	ms.lastUpdated[key] = time.Now()
	if ms.notifier != nil {
		ms.notifier()
	}
}

func (ms *MemoryStore) Clear() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.fundingRates = make(map[portfolio.Asset]store.FundingRateMap)
	ms.historicalFundingRates = make(map[portfolio.Asset]store.HistoricalFundingMap)
	ms.orderBooks = make(map[portfolio.Asset]store.OrderBookMap)
	ms.assetPrices = make(map[portfolio.Asset]store.PriceMap)
	ms.klines = make(map[portfolio.Asset]store.KlineMap)
	ms.lastUpdated = make(store.LastUpdatedMap)
}
