package paradex

import (
	"fmt"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
)

func (p *Paradex) FetchCurrentFundingRates() (map[portfolio.Asset]connector.FundingRate, error) {
	return nil, fmt.Errorf("current funding rates not needed for MM strategy")
}

func (p *Paradex) FetchFundingRate(asset portfolio.Asset) (*connector.FundingRate, error) {
	return nil, fmt.Errorf("funding rate not needed for MM strategy")

}

func (p *Paradex) FetchHistoricalFundingRates(asset portfolio.Asset, startTime, endTime int64) ([]connector.HistoricalFundingRate, error) {
	return nil, fmt.Errorf("historical funding rates not needed for MM strategy")
}
