package data

import (
	"github.com/sonirico/go-hyperliquid"
)

type MarketDataService struct {
	info *hyperliquid.Info
}

var millisecondsPerSecond = int64(1000)

func NewMarketDataService(info *hyperliquid.Info) *MarketDataService {
	return &MarketDataService{info: info}
}

func (m *MarketDataService) GetAllMids() (map[string]string, error) {
	return m.info.AllMids()
}

func (m *MarketDataService) GetL2Book(coin string) (*hyperliquid.L2Book, error) {
	return m.info.L2Snapshot(coin)
}

func (m *MarketDataService) GetCandles(coin, interval string, startTime, endTime int64) ([]hyperliquid.Candle, error) {
	return m.info.CandlesSnapshot(coin, interval, startTime*millisecondsPerSecond, endTime*millisecondsPerSecond)
}

func (m *MarketDataService) GetMeta() (*hyperliquid.Meta, error) {
	return m.info.Meta()
}

func (m *MarketDataService) GetSpotMeta() (*hyperliquid.SpotMeta, error) {
	return m.info.SpotMeta()
}

func (m *MarketDataService) GetMetaAndAssetCtxs() (map[string]any, error) {
	return m.info.MetaAndAssetCtxs()
}

func (m *MarketDataService) GetSpotMetaAndAssetCtxs() (map[string]any, error) {
	return m.info.SpotMetaAndAssetCtxs()
}

func (m *MarketDataService) NameToAsset(name string) int {
	return m.info.NameToAsset(name)
}
