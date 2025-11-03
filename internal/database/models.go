package database

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PluginMetadata represents a loaded plugin's metadata
type PluginMetadata struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	Name        string          `db:"name" json:"name"`
	Description string          `db:"description" json:"description"`
	PluginPath  string          `db:"plugin_path" json:"plugin_path"`
	RiskLevel   string          `db:"risk_level" json:"risk_level"`
	Type        string          `db:"type" json:"type"`
	Version     string          `db:"version" json:"version"`
	Parameters  ParameterDefMap `db:"parameters" json:"parameters"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
	DeletedAt   *time.Time      `db:"deleted_at" json:"deleted_at,omitempty"`
	CreatedBy   string          `db:"created_by" json:"created_by"`
}

// ParameterDef defines a strategy parameter
type ParameterDef struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // decimal, int, string, bool
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"`
	Required    bool        `json:"required"`
	Min         interface{} `json:"min,omitempty"`
	Max         interface{} `json:"max,omitempty"`
}

// ParameterDefMap is a map of parameter definitions
type ParameterDefMap map[string]ParameterDef

// Value implements driver.Valuer for database storage
func (p ParameterDefMap) Value() (driver.Value, error) {
	if p == nil {
		return json.Marshal(map[string]ParameterDef{})
	}
	return json.Marshal(p)
}

// Scan implements sql.Scanner for database retrieval
func (p *ParameterDefMap) Scan(value interface{}) error {
	if value == nil {
		*p = make(ParameterDefMap)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, p)
}

// StrategyConfig represents a strategy configuration
type StrategyConfig struct {
	ID         uuid.UUID       `db:"id" json:"id"`
	PluginID   uuid.UUID       `db:"plugin_id" json:"plugin_id"`
	Version    int             `db:"version" json:"version"`
	ConfigData ConfigData      `db:"config_data" json:"config_data"`
	CreatedAt  time.Time       `db:"created_at" json:"created_at"`
	CreatedBy  string          `db:"created_by" json:"created_by"`
}

// ConfigData is a JSON map of configuration values
type ConfigData map[string]interface{}

// Value implements driver.Valuer for database storage
func (c ConfigData) Value() (driver.Value, error) {
	if c == nil {
		return json.Marshal(map[string]interface{}{})
	}
	return json.Marshal(c)
}

// Scan implements sql.Scanner for database retrieval
func (c *ConfigData) Scan(value interface{}) error {
	if value == nil {
		*c = make(ConfigData)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, c)
}

// StrategyRun represents a strategy execution run
type StrategyRun struct {
	ID           uuid.UUID        `db:"id" json:"id"`
	PluginID     uuid.UUID        `db:"plugin_id" json:"plugin_id"`
	ConfigID     *uuid.UUID       `db:"config_id" json:"config_id,omitempty"`
	Status       string           `db:"status" json:"status"` // running, stopped, error, completed
	StartTime    time.Time        `db:"start_time" json:"start_time"`
	EndTime      *time.Time       `db:"end_time" json:"end_time,omitempty"`
	TotalSignals int64            `db:"total_signals" json:"total_signals"`
	TotalTrades  int64            `db:"total_trades" json:"total_trades"`
	ProfitLoss   decimal.Decimal  `db:"profit_loss" json:"profit_loss"`
	ErrorCount   int64            `db:"error_count" json:"error_count"`
	ErrorMessage *string          `db:"error_message" json:"error_message,omitempty"`
	CPUUsage     float64          `db:"cpu_usage" json:"cpu_usage"`
	MemoryUsage  int64            `db:"memory_usage" json:"memory_usage"`
	CreatedAt    time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time        `db:"updated_at" json:"updated_at"`
}

// TradingSignal represents a generated trading signal
type TradingSignal struct {
	ID        uuid.UUID       `db:"id" json:"id"`
	RunID     uuid.UUID       `db:"run_id" json:"run_id"`
	SignalType string         `db:"signal_type" json:"signal_type"` // BUY, SELL, HOLD
	Asset     string          `db:"asset" json:"asset"`
	Exchange  string          `db:"exchange" json:"exchange"`
	Quantity  decimal.Decimal `db:"quantity" json:"quantity"`
	Price     decimal.Decimal `db:"price" json:"price"`
	Timestamp time.Time       `db:"timestamp" json:"timestamp"`
	Executed  bool            `db:"executed" json:"executed"`
	OrderID   *string         `db:"order_id" json:"order_id,omitempty"`
}

// ExecutionLog represents a log entry for strategy execution
type ExecutionLog struct {
	ID        uuid.UUID `db:"id" json:"id"`
	RunID     uuid.UUID `db:"run_id" json:"run_id"`
	Level     string    `db:"level" json:"level"` // info, warning, error
	Message   string    `db:"message" json:"message"`
	Metadata  LogMetadata `db:"metadata" json:"metadata,omitempty"`
	Timestamp time.Time `db:"timestamp" json:"timestamp"`
}

// LogMetadata is a JSON map of log metadata
type LogMetadata map[string]interface{}

// Value implements driver.Valuer for database storage
func (l LogMetadata) Value() (driver.Value, error) {
	if l == nil {
		return json.Marshal(map[string]interface{}{})
	}
	return json.Marshal(l)
}

// Scan implements sql.Scanner for database retrieval
func (l *LogMetadata) Scan(value interface{}) error {
	if value == nil {
		*l = make(LogMetadata)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, l)
}

// RunStatus constants
const (
	RunStatusRunning   = "running"
	RunStatusStopped   = "stopped"
	RunStatusError     = "error"
	RunStatusCompleted = "completed"
)

// SignalType constants
const (
	SignalTypeBuy  = "BUY"
	SignalTypeSell = "SELL"
	SignalTypeHold = "HOLD"
)

// LogLevel constants
const (
	LogLevelInfo    = "info"
	LogLevelWarning = "warning"
	LogLevelError   = "error"
)
