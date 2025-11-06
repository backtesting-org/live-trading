package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// hasQuery reports whether the DSN already has a query string
func hasQuery(dsn string) bool {
    for i := 0; i < len(dsn); i++ {
        if dsn[i] == '?' {
            return true
        }
    }
    return false
}

// containsParam reports whether the DSN contains a given parameter key
func containsParam(dsn, key string) bool {
    // naive but sufficient: look for key= substring
    needle := key + "="
    return len(dsn) >= len(needle) && (indexOf(dsn, needle) >= 0)
}

func indexOf(s, sub string) int {
    // simple substring search
    for i := 0; i+len(sub) <= len(s); i++ {
        if s[i:i+len(sub)] == sub {
            return i
        }
    }
    return -1
}

// Repository provides database operations
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new database repository
func NewRepository(connectionString string) (*Repository, error) {
    // Ensure simple protocol to avoid server-side prepared statements
    dsn := connectionString
    if dsn != "" && !containsParam(dsn, "prefer_simple_protocol") {
        sep := "?"
        if hasQuery(dsn) { sep = "&" }
        dsn = dsn + sep + "prefer_simple_protocol=true"
    }

    db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
    db.SetMaxOpenConns(25)
    // Avoid reusing connections to prevent unnamed prepared statement mismatches
    db.SetMaxIdleConns(0)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(10 * time.Minute)

	repo := &Repository{db: db}

    // No need to clear prepared statements when using simple protocol

	return repo, nil
}

// Close closes the database connection
func (r *Repository) Close() error {
	return r.db.Close()
}

// Ping verifies the database connection
func (r *Repository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

// clearPreparedStatementCache clears all PostgreSQL prepared statements
// This prevents "bind message supplies X parameters, but prepared statement requires Y" errors
// that occur when prepared statements are cached with wrong parameter counts
func (r *Repository) clearPreparedStatementCache(ctx context.Context) error {
    // No-op under pgx simple protocol; kept for compatibility
    return nil
}

// RunMigrations executes database migrations
func (r *Repository) RunMigrations(ctx context.Context, migrationSQL string) error {
	_, err := r.db.ExecContext(ctx, migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

// === Plugin Metadata Operations ===

// CreatePlugin creates a new plugin metadata entry
func (r *Repository) CreatePlugin(ctx context.Context, plugin *PluginMetadata) error {
	query := `
		INSERT INTO plugin_metadata (
			id, name, description, plugin_path, risk_level, type, version, parameters, created_by
		) VALUES (
			:id, :name, :description, :plugin_path, :risk_level, :type, :version, :parameters, :created_by
		)
		RETURNING created_at, updated_at
	`

    rows, err := r.db.NamedQueryContext(ctx, query, plugin)
    if err != nil {
        return fmt.Errorf("failed to create plugin: %w", err)
    }
    defer rows.Close()
    if rows.Next() {
        if err := rows.StructScan(plugin); err != nil {
            return fmt.Errorf("failed to scan created plugin: %w", err)
        }
    }
    return nil
}

// GetPlugin retrieves a plugin by ID
func (r *Repository) GetPlugin(ctx context.Context, id uuid.UUID) (*PluginMetadata, error) {
	var plugin PluginMetadata
	query := `
		SELECT id, name, description, plugin_path, risk_level, type, version,
		       parameters, created_at, updated_at, deleted_at, created_by
		FROM plugin_metadata WHERE id = $1 AND deleted_at IS NULL
	`

	err := r.db.GetContext(ctx, &plugin, query, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("plugin not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin: %w", err)
	}

	return &plugin, nil
}

// GetPluginByName retrieves a plugin by name
func (r *Repository) GetPluginByName(ctx context.Context, name string) (*PluginMetadata, error) {
	var plugin PluginMetadata
	query := `
		SELECT id, name, description, plugin_path, risk_level, type, version,
		       parameters, created_at, updated_at, deleted_at, created_by
		FROM plugin_metadata WHERE name = $1 AND deleted_at IS NULL
	`

	err := r.db.GetContext(ctx, &plugin, query, name)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("plugin not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin: %w", err)
	}

	return &plugin, nil
}

// ListPlugins retrieves all active plugins
func (r *Repository) ListPlugins(ctx context.Context, limit, offset int) ([]PluginMetadata, error) {
	var plugins []PluginMetadata
	query := `
		SELECT id, name, description, plugin_path, risk_level, type, version,
		       parameters, created_at, updated_at, deleted_at, created_by
		FROM plugin_metadata
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	err := r.db.SelectContext(ctx, &plugins, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list plugins: %w", err)
	}

	return plugins, nil
}

// UpdatePlugin updates plugin metadata
func (r *Repository) UpdatePlugin(ctx context.Context, plugin *PluginMetadata) error {
	query := `
		UPDATE plugin_metadata SET
			description = :description,
			plugin_path = :plugin_path,
			risk_level = :risk_level,
			type = :type,
			version = :version,
			parameters = :parameters
		WHERE id = :id AND deleted_at IS NULL
	`

	result, err := r.db.NamedExecContext(ctx, query, plugin)
	if err != nil {
		return fmt.Errorf("failed to update plugin: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("plugin not found or already deleted")
	}

	return nil
}

// DeletePlugin soft deletes a plugin
func (r *Repository) DeletePlugin(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE plugin_metadata SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete plugin: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("plugin not found or already deleted")
	}

	return nil
}

// === Strategy Config Operations ===

// CreateConfig creates a new strategy configuration
func (r *Repository) CreateConfig(ctx context.Context, config *StrategyConfig) error {
	query := `
		INSERT INTO strategy_configs (
			id, plugin_id, version, config_data, created_by
		) VALUES (
			:id, :plugin_id, :version, :config_data, :created_by
		)
		RETURNING created_at
	`

    rows, err := r.db.NamedQueryContext(ctx, query, config)
    if err != nil {
        return fmt.Errorf("failed to create config: %w", err)
    }
    defer rows.Close()
    if rows.Next() {
        if err := rows.StructScan(config); err != nil {
            return fmt.Errorf("failed to scan created config: %w", err)
        }
    }
    return nil
}

// GetLatestConfig retrieves the latest configuration for a plugin
func (r *Repository) GetLatestConfig(ctx context.Context, pluginID uuid.UUID) (*StrategyConfig, error) {
	var config StrategyConfig
	query := `
		SELECT id, plugin_id, version, config_data, created_at, created_by
		FROM strategy_configs
		WHERE plugin_id = $1
		ORDER BY version DESC
		LIMIT 1
	`

	err := r.db.GetContext(ctx, &config, query, pluginID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no config found for plugin")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	return &config, nil
}

// GetConfigByVersion retrieves a specific version of configuration
func (r *Repository) GetConfigByVersion(ctx context.Context, pluginID uuid.UUID, version int) (*StrategyConfig, error) {
	var config StrategyConfig
	query := `
		SELECT id, plugin_id, version, config_data, created_at, created_by
		FROM strategy_configs WHERE plugin_id = $1 AND version = $2
	`

	err := r.db.GetContext(ctx, &config, query, pluginID, version)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("config version not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	return &config, nil
}

// ListConfigs retrieves all configurations for a plugin
func (r *Repository) ListConfigs(ctx context.Context, pluginID uuid.UUID) ([]StrategyConfig, error) {
	var configs []StrategyConfig
	query := `
		SELECT id, plugin_id, version, config_data, created_at, created_by
		FROM strategy_configs WHERE plugin_id = $1 ORDER BY version DESC
	`

	err := r.db.SelectContext(ctx, &configs, query, pluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to list configs: %w", err)
	}

	return configs, nil
}

// === Strategy Run Operations ===

// CreateRun creates a new strategy run
func (r *Repository) CreateRun(ctx context.Context, run *StrategyRun) error {
	query := `
		INSERT INTO strategy_runs (
			id, plugin_id, config_id, status, start_time
		) VALUES (
			:id, :plugin_id, :config_id, :status, :start_time
		)
		RETURNING created_at, updated_at
	`

    rows, err := r.db.NamedQueryContext(ctx, query, run)
    if err != nil {
        return fmt.Errorf("failed to create run: %w", err)
    }
    defer rows.Close()
    if rows.Next() {
        if err := rows.StructScan(run); err != nil {
            return fmt.Errorf("failed to scan created run: %w", err)
        }
    }
    return nil
}

// GetRun retrieves a run by ID
func (r *Repository) GetRun(ctx context.Context, id uuid.UUID) (*StrategyRun, error) {
	var run StrategyRun
	query := `
		SELECT id, plugin_id, config_id, status, start_time, end_time,
		       total_signals, total_trades, profit_loss, error_count,
		       error_message, cpu_usage, memory_usage, created_at, updated_at
		FROM strategy_runs WHERE id = $1
	`

	err := r.db.GetContext(ctx, &run, query, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("run not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get run: %w", err)
	}

	return &run, nil
}

// ListRunsByPlugin retrieves all runs for a plugin
func (r *Repository) ListRunsByPlugin(ctx context.Context, pluginID uuid.UUID, limit, offset int) ([]StrategyRun, error) {
    var runs []StrategyRun
    query := `
        SELECT id, plugin_id, config_id, status, start_time, end_time,
               total_signals, total_trades, profit_loss, error_count,
               error_message, cpu_usage, memory_usage, created_at, updated_at
        FROM strategy_runs
        WHERE plugin_id = $1
        ORDER BY start_time DESC
        LIMIT $2 OFFSET $3
    `

    if err := r.db.SelectContext(ctx, &runs, query, pluginID, limit, offset); err != nil {
        return nil, fmt.Errorf("failed to list runs: %w", err)
    }

    return runs, nil
}

// GetActiveRun retrieves the currently active run for a plugin
func (r *Repository) GetActiveRun(ctx context.Context, pluginID uuid.UUID) (*StrategyRun, error) {
	var run StrategyRun
	query := `
		SELECT id, plugin_id, config_id, status, start_time, end_time,
		       total_signals, total_trades, profit_loss, error_count,
		       error_message, cpu_usage, memory_usage, created_at, updated_at
		FROM strategy_runs
		WHERE plugin_id = $1 AND status = $2
		ORDER BY start_time DESC
		LIMIT 1
	`

	err := r.db.GetContext(ctx, &run, query, pluginID, RunStatusRunning)
	if err == sql.ErrNoRows {
		return nil, nil // No active run is not an error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active run: %w", err)
	}

	return &run, nil
}

// UpdateRun updates a strategy run
func (r *Repository) UpdateRun(ctx context.Context, run *StrategyRun) error {
	query := `
		UPDATE strategy_runs SET
			status = :status,
			end_time = :end_time,
			total_signals = :total_signals,
			total_trades = :total_trades,
			profit_loss = :profit_loss,
			error_count = :error_count,
			error_message = :error_message,
			cpu_usage = :cpu_usage,
			memory_usage = :memory_usage
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, run)
	if err != nil {
		return fmt.Errorf("failed to update run: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("run not found")
	}

	return nil
}

// === Trading Signal Operations ===

// CreateSignal creates a new trading signal
func (r *Repository) CreateSignal(ctx context.Context, signal *TradingSignal) error {
	query := `
		INSERT INTO trading_signals (
			id, run_id, signal_type, asset, exchange, quantity, price, timestamp, executed, order_id
		) VALUES (
			:id, :run_id, :signal_type, :asset, :exchange, :quantity, :price, :timestamp, :executed, :order_id
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, signal)
	if err != nil {
		return fmt.Errorf("failed to create signal: %w", err)
	}

	return nil
}

// ListSignalsByRun retrieves all signals for a run
func (r *Repository) ListSignalsByRun(ctx context.Context, runID uuid.UUID, limit, offset int) ([]TradingSignal, error) {
	var signals []TradingSignal
	query := `
		SELECT id, run_id, signal_type, asset, exchange, quantity, price,
		       timestamp, executed, order_id
		FROM trading_signals
		WHERE run_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	err := r.db.SelectContext(ctx, &signals, query, runID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list signals: %w", err)
	}

	return signals, nil
}

// UpdateSignal updates a trading signal
func (r *Repository) UpdateSignal(ctx context.Context, signal *TradingSignal) error {
	query := `
		UPDATE trading_signals SET
			executed = :executed,
			order_id = :order_id
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, signal)
	if err != nil {
		return fmt.Errorf("failed to update signal: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("signal not found")
	}

	return nil
}

// === Execution Log Operations ===

// CreateLog creates a new execution log entry
func (r *Repository) CreateLog(ctx context.Context, log *ExecutionLog) error {
	query := `
		INSERT INTO execution_logs (
			id, run_id, level, message, metadata, timestamp
		) VALUES (
			:id, :run_id, :level, :message, :metadata, :timestamp
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, log)
	if err != nil {
		return fmt.Errorf("failed to create log: %w", err)
	}

	return nil
}

// ListLogsByRun retrieves all logs for a run
func (r *Repository) ListLogsByRun(ctx context.Context, runID uuid.UUID, limit, offset int) ([]ExecutionLog, error) {
	var logs []ExecutionLog
	query := `
		SELECT id, run_id, level, message, metadata, timestamp
		FROM execution_logs
		WHERE run_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	err := r.db.SelectContext(ctx, &logs, query, runID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list logs: %w", err)
	}

	return logs, nil
}

// GetRunStats retrieves statistics for a run
func (r *Repository) GetRunStats(ctx context.Context, runID uuid.UUID) (map[string]interface{}, error) {
	var stats struct {
		TotalSignals   int64   `db:"total_signals"`
		ExecutedSignals int64  `db:"executed_signals"`
		BuySignals     int64   `db:"buy_signals"`
		SellSignals    int64   `db:"sell_signals"`
		ErrorCount     int64   `db:"error_count"`
	}

	query := `
		SELECT
			COUNT(*) as total_signals,
			COUNT(*) FILTER (WHERE executed = true) as executed_signals,
			COUNT(*) FILTER (WHERE signal_type = 'BUY') as buy_signals,
			COUNT(*) FILTER (WHERE signal_type = 'SELL') as sell_signals,
			(SELECT error_count FROM strategy_runs WHERE id = $1) as error_count
		FROM trading_signals
		WHERE run_id = $1
	`

	err := r.db.GetContext(ctx, &stats, query, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get run stats: %w", err)
	}

	return map[string]interface{}{
		"total_signals":    stats.TotalSignals,
		"executed_signals": stats.ExecutedSignals,
		"buy_signals":      stats.BuySignals,
		"sell_signals":     stats.SellSignals,
		"error_count":      stats.ErrorCount,
	}, nil
}
