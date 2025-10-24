package connection

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/backtesting-org/live-trading/external/websocket/security"
)

type ReconnectStrategy interface {
	NextDelay(attempt int) time.Duration
	ShouldReconnect(attempt int, err error) bool
	Reset()
}

type ExponentialBackoffStrategy struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	MaxAttempts  int
	Multiplier   float64
	Jitter       bool
	randSource   *rand.Rand
	mutex        sync.Mutex
}

func NewExponentialBackoffStrategy(initialDelay, maxDelay time.Duration, maxAttempts int) *ExponentialBackoffStrategy {
	return &ExponentialBackoffStrategy{
		InitialDelay: initialDelay,
		MaxDelay:     maxDelay,
		MaxAttempts:  maxAttempts,
		Multiplier:   2.0,
		Jitter:       true,
		randSource:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (ebs *ExponentialBackoffStrategy) NextDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return ebs.InitialDelay
	}

	delay := float64(ebs.InitialDelay) * math.Pow(ebs.Multiplier, float64(attempt-1))

	if delay > float64(ebs.MaxDelay) {
		delay = float64(ebs.MaxDelay)
	}

	if ebs.Jitter {
		ebs.mutex.Lock()
		jitterFactor := 2*ebs.randSource.Float64() - 1
		ebs.mutex.Unlock()

		jitter := delay * 0.1 * jitterFactor
		delay += jitter

		if delay < 0 {
			delay = float64(ebs.InitialDelay)
		}
	}

	return time.Duration(delay)
}

func (ebs *ExponentialBackoffStrategy) ShouldReconnect(attempt int, err error) bool {
	return attempt < ebs.MaxAttempts
}

func (ebs *ExponentialBackoffStrategy) Reset() {
}

type ReconnectManager struct {
	connectionManager *ConnectionManager
	strategy          ReconnectStrategy
	logger            security.Logger

	isReconnecting bool
	reconnectMutex sync.Mutex
	currentAttempt int

	onReconnectStart   func(attempt int)
	onReconnectFail    func(attempt int, err error)
	onReconnectSuccess func(attempt int)
}

func NewReconnectManager(
	connectionManager *ConnectionManager,
	strategy ReconnectStrategy,
	logger security.Logger,
) *ReconnectManager {
	return &ReconnectManager{
		connectionManager: connectionManager,
		strategy:          strategy,
		logger:            logger,
	}
}

func (rm *ReconnectManager) SetCallbacks(
	onStart func(int),
	onFail func(int, error),
	onSuccess func(int),
) {
	rm.onReconnectStart = onStart
	rm.onReconnectFail = onFail
	rm.onReconnectSuccess = onSuccess
}

func (rm *ReconnectManager) StartReconnection(ctx context.Context) {
	rm.reconnectMutex.Lock()
	defer rm.reconnectMutex.Unlock()

	if rm.isReconnecting {
		rm.logger.Debug("Reconnection already in progress")
		return
	}

	rm.isReconnecting = true
	rm.currentAttempt = 0

	go rm.reconnectLoop(ctx)
}

func (rm *ReconnectManager) reconnectLoop(ctx context.Context) {
	defer func() {
		rm.reconnectMutex.Lock()
		rm.isReconnecting = false
		rm.reconnectMutex.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			rm.logger.Debug("Reconnection cancelled by context")
			return
		default:
			rm.currentAttempt++

			if !rm.strategy.ShouldReconnect(rm.currentAttempt, nil) {
				rm.logger.Error("Max reconnection attempts reached: %d", rm.currentAttempt-1)
				if rm.onReconnectFail != nil {
					rm.onReconnectFail(rm.currentAttempt-1, fmt.Errorf("max attempts reached"))
				}
				return
			}

			delay := rm.strategy.NextDelay(rm.currentAttempt)
			rm.logger.Debug("Attempting reconnection %d after %v delay", rm.currentAttempt, delay)

			if rm.onReconnectStart != nil {
				rm.onReconnectStart(rm.currentAttempt)
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}

			err := rm.connectionManager.Connect(ctx)
			if err == nil {
				rm.logger.Info("Reconnection successful after %d attempts", rm.currentAttempt)
				if rm.onReconnectSuccess != nil {
					rm.onReconnectSuccess(rm.currentAttempt)
				}
				rm.strategy.Reset()
				return
			}

			rm.logger.Debug("Reconnection attempt %d failed: %v", rm.currentAttempt, err)
			if rm.onReconnectFail != nil {
				rm.onReconnectFail(rm.currentAttempt, err)
			}
		}
	}
}

func (rm *ReconnectManager) IsReconnecting() bool {
	rm.reconnectMutex.Lock()
	defer rm.reconnectMutex.Unlock()
	return rm.isReconnecting
}

func (rm *ReconnectManager) GetCurrentAttempt() int {
	rm.reconnectMutex.Lock()
	defer rm.reconnectMutex.Unlock()
	return rm.currentAttempt
}

func (rm *ReconnectManager) Stop() {
	rm.reconnectMutex.Lock()
	defer rm.reconnectMutex.Unlock()
	rm.isReconnecting = false
}
