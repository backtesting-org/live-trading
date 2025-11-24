package data

import (
	"github.com/sonirico/go-hyperliquid"
)

func (m *MarketDataService) GetUserState(user string) (hyperliquid.UserState, error) {
	return m.info.SpotUserState(user)
}

func (m *MarketDataService) GetOpenOrders(user string) ([]hyperliquid.OpenOrder, error) {
	return m.info.OpenOrders(user)
}

func (m *MarketDataService) GetUserFills(user string) ([]hyperliquid.Fill, error) {
	return m.info.UserFills(user)
}
