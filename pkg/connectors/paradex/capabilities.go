package paradex

import (
	"fmt"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/shopspring/decimal"
)

func (p *paradex) GetConnectorInfo() *connector.Info {
	return &connector.Info{
		Name:             connector.Paradex,
		TradingEnabled:   p.SupportsTradingOperations(),
		WebSocketEnabled: p.SupportsRealTimeData(),
		MaxLeverage:      decimal.NewFromFloat(10.0),
		SupportedOrderTypes: []connector.OrderType{
			connector.OrderTypeLimit,
			connector.OrderTypeMarket,
		},
		QuoteCurrency: "USD",
	}
}

func (p *paradex) GetPerpSymbol(symbol portfolio.Asset) string {
	return fmt.Sprintf("%s-USD-PERP", symbol.Symbol())
}

func (p *paradex) SupportsTradingOperations() bool {
	return true
}

func (p *paradex) SupportsRealTimeData() bool {
	return true
}

func (p *paradex) SupportsFundingRates() bool {
	return true
}

func (p *paradex) SupportsPerpetuals() bool {
	return true
}

func (p *paradex) SupportsSpot() bool {
	return false
}
