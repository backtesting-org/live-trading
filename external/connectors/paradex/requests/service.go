package requests

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/live-trading/external/connectors/paradex/adaptor"
)

type Service struct {
	client *adaptor.Client
	logger logging.ApplicationLogger
}

func NewService(client *adaptor.Client, logger logging.ApplicationLogger) *Service {
	return &Service{
		client: client,
		logger: logger,
	}
}
