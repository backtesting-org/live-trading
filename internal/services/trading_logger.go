package services

import (
	"fmt"

	"go.uber.org/zap"
)

// TradingLoggerAdapter wraps zap.Logger to implement the TradingLogger interface
type TradingLoggerAdapter struct {
	logger *zap.Logger
}

// NewTradingLogger creates a new trading logger adapter
func NewTradingLogger(logger *zap.Logger) *TradingLoggerAdapter {
	return &TradingLoggerAdapter{
		logger: logger,
	}
}

// MarketCondition logs market condition events
func (tl *TradingLoggerAdapter) MarketCondition(msg string, args ...interface{}) {
	tl.logger.Info(
		"[MARKET] "+msg,
		zap.Any("args", args),
	)
}

// Opportunity logs trading opportunities
func (tl *TradingLoggerAdapter) Opportunity(strategy, asset, msg string, args ...interface{}) {
	tl.logger.Info(
		"[OPPORTUNITY] "+msg,
		zap.String("strategy", strategy),
		zap.String("asset", asset),
		zap.Any("args", args),
	)
}

// Success logs successful trading operations
func (tl *TradingLoggerAdapter) Success(strategy, asset, msg string, args ...interface{}) {
	tl.logger.Info(
		"[SUCCESS] "+msg,
		zap.String("strategy", strategy),
		zap.String("asset", asset),
		zap.Any("args", args),
	)
}

// Failed logs failed trading operations
func (tl *TradingLoggerAdapter) Failed(strategy, asset, msg string, args ...interface{}) {
	tl.logger.Error(
		"[FAILED] "+msg,
		zap.String("strategy", strategy),
		zap.String("asset", asset),
		zap.Any("args", args),
	)
}

// OrderLifecycle logs order lifecycle events
func (tl *TradingLoggerAdapter) OrderLifecycle(msg, asset string, args ...interface{}) {
	tl.logger.Info(
		"[ORDER] "+msg,
		zap.String("asset", asset),
		zap.Any("args", args),
	)
}

// DataCollection logs data collection events
func (tl *TradingLoggerAdapter) DataCollection(exchange, msg string, args ...interface{}) {
	tl.logger.Debug(
		"[DATA] "+msg,
		zap.String("exchange", exchange),
		zap.Any("args", args),
	)
}

// Debug logs debug messages for trading events
func (tl *TradingLoggerAdapter) Debug(strategy, asset, msg string, args ...interface{}) {
	tl.logger.Debug(
		"[DEBUG] "+msg,
		zap.String("strategy", strategy),
		zap.String("asset", asset),
		zap.Any("args", args),
	)
}

// Info logs general trading information
func (tl *TradingLoggerAdapter) Info(msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	tl.logger.Info("[TRADING] " + msg)
}
