package paradex

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
)

func (p *Paradex) FetchAvailableSpotAssets() ([]portfolio.Asset, error) {
	return []portfolio.Asset{}, nil
}

func (p *Paradex) FetchAvailablePerpetualAssets() ([]portfolio.Asset, error) {
	markets, err := p.paradexService.GetMarkets(p.ctx)
	if err != nil {
		return nil, err
	}

	var assets []portfolio.Asset
	for _, market := range markets {
		if market.AssetKind != "PERP" {
			continue
		}
		asset := portfolio.NewAsset(market.BaseCurrency)
		assets = append(assets, asset)
	}

	return assets, nil
}
