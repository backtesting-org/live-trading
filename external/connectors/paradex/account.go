package paradex

import (
	"context"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/shopspring/decimal"
)

func (p *Paradex) GetAccountBalance() (*connector.AccountBalance, error) {
	account, err := p.paradexService.GetAccount(p.ctx)
	if err != nil {
		return nil, err
	}

	// Parse all needed fields
	accountValue, _ := decimal.NewFromString(account.AccountValue)
	totalCollateral, _ := decimal.NewFromString(account.TotalCollateral)
	freeCollateral, _ := decimal.NewFromString(account.FreeCollateral)
	initialMargin, _ := decimal.NewFromString(account.InitialMarginRequirement)
	currency := account.SettlementAsset
	if currency == "" {
		currency = "USD"
	}

	// Calculations
	usedMargin := initialMargin
	unrealizedPnL := accountValue.Sub(totalCollateral)

	updatedAt := time.Now()
	if account.UpdatedAt > 0 {
		updatedAt = time.UnixMilli(account.UpdatedAt)
	}

	return &connector.AccountBalance{
		TotalBalance:     accountValue,   // account_value (includes unrealized PnL)
		AvailableBalance: freeCollateral, // free_collateral
		UsedMargin:       usedMargin,     // initial_margin_requirement
		UnrealizedPnL:    unrealizedPnL,  // account_value - total_collateral
		Currency:         currency,
		UpdatedAt:        updatedAt,
	}, nil
}

func (p *Paradex) GetPositions() ([]connector.Position, error) {
	positionsResp, err := p.paradexService.GetUserPositions(p.ctx) // returns *models.ResponsesGetPositionsResp
	if err != nil {
		return nil, err
	}

	var result []connector.Position
	for _, pos := range positionsResp.Results {
		size, _ := decimal.NewFromString(pos.Size)
		entryPrice, _ := decimal.NewFromString(pos.AverageEntryPrice)
		unrealizedPnL, _ := decimal.NewFromString(pos.UnrealizedPnl)

		// MarkPrice is not in the Paradex API, so set to zero
		var markPrice decimal.Decimal

		realizedPnL, _ := decimal.NewFromString(pos.RealizedPositionalPnl)
		updatedAt := time.UnixMilli(pos.LastUpdatedAt)

		result = append(result, connector.Position{
			Symbol:        portfolio.NewAsset(pos.Market),
			Exchange:      p.GetConnectorInfo().Name,
			Side:          connector.OrderSide(pos.Side),
			Size:          size,
			EntryPrice:    entryPrice,
			MarkPrice:     markPrice,
			UnrealizedPnL: unrealizedPnL,
			RealizedPnL:   realizedPnL,
			UpdatedAt:     updatedAt,
		})
	}

	return result, nil
}

// GetSubAccounts returns all sub-accounts for the current account
func (p *Paradex) GetSubAccounts(ctx context.Context) (interface{}, error) {
	return p.paradexService.GetSubAccounts(ctx)
}

// GetAccountInfo returns account information
func (p *Paradex) GetAccountInfo(ctx context.Context) (interface{}, error) {
	return p.paradexService.GetAccountInfo(ctx)
}
