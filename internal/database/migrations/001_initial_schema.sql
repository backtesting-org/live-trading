-- Initial schema for live-trading platform

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Plugin metadata table
CREATE TABLE IF NOT EXISTS plugin_metadata (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    plugin_path VARCHAR(512) NOT NULL,
    risk_level VARCHAR(50) NOT NULL CHECK (risk_level IN ('low', 'medium', 'high')),
    type VARCHAR(50) NOT NULL,
    version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    parameters JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    created_by VARCHAR(255) NOT NULL DEFAULT 'system'
);

-- Create index on plugin name for fast lookups
CREATE INDEX IF NOT EXISTS idx_plugin_metadata_name ON plugin_metadata(name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_plugin_metadata_created_at ON plugin_metadata(created_at DESC);

-- Strategy configurations table (immutable snapshots)
CREATE TABLE IF NOT EXISTS strategy_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    plugin_id UUID NOT NULL REFERENCES plugin_metadata(id) ON DELETE CASCADE,
    version INT NOT NULL DEFAULT 1,
    config_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255) NOT NULL DEFAULT 'system',
    UNIQUE(plugin_id, version)
);

-- Create index on plugin_id for fast lookups
CREATE INDEX IF NOT EXISTS idx_strategy_configs_plugin_id ON strategy_configs(plugin_id);
CREATE INDEX IF NOT EXISTS idx_strategy_configs_created_at ON strategy_configs(created_at DESC);

-- Strategy execution runs table
CREATE TABLE IF NOT EXISTS strategy_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    plugin_id UUID NOT NULL REFERENCES plugin_metadata(id) ON DELETE CASCADE,
    config_id UUID NULL REFERENCES strategy_configs(id) ON DELETE SET NULL,
    status VARCHAR(50) NOT NULL CHECK (status IN ('running', 'stopped', 'error', 'completed')),
    start_time TIMESTAMP NOT NULL DEFAULT NOW(),
    end_time TIMESTAMP NULL,
    total_signals BIGINT NOT NULL DEFAULT 0,
    total_trades BIGINT NOT NULL DEFAULT 0,
    profit_loss DECIMAL(20,8) NOT NULL DEFAULT 0,
    error_count BIGINT NOT NULL DEFAULT 0,
    error_message TEXT NULL,
    cpu_usage FLOAT NOT NULL DEFAULT 0,
    memory_usage BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes for strategy runs
CREATE INDEX IF NOT EXISTS idx_strategy_runs_plugin_id ON strategy_runs(plugin_id);
CREATE INDEX IF NOT EXISTS idx_strategy_runs_status ON strategy_runs(status);
CREATE INDEX IF NOT EXISTS idx_strategy_runs_start_time ON strategy_runs(start_time DESC);
CREATE INDEX IF NOT EXISTS idx_strategy_runs_created_at ON strategy_runs(created_at DESC);

-- Trading signals table
CREATE TABLE IF NOT EXISTS trading_signals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID NOT NULL REFERENCES strategy_runs(id) ON DELETE CASCADE,
    signal_type VARCHAR(50) NOT NULL CHECK (signal_type IN ('BUY', 'SELL', 'HOLD')),
    asset VARCHAR(50) NOT NULL,
    exchange VARCHAR(50) NOT NULL,
    quantity DECIMAL(20,8) NOT NULL,
    price DECIMAL(20,8) NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    executed BOOLEAN NOT NULL DEFAULT FALSE,
    order_id VARCHAR(255) NULL
);

-- Create indexes for trading signals
CREATE INDEX IF NOT EXISTS idx_trading_signals_run_id ON trading_signals(run_id);
CREATE INDEX IF NOT EXISTS idx_trading_signals_timestamp ON trading_signals(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_trading_signals_asset ON trading_signals(asset);
CREATE INDEX IF NOT EXISTS idx_trading_signals_executed ON trading_signals(executed);

-- Execution logs table
CREATE TABLE IF NOT EXISTS execution_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID NOT NULL REFERENCES strategy_runs(id) ON DELETE CASCADE,
    level VARCHAR(50) NOT NULL CHECK (level IN ('info', 'warning', 'error')),
    message TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes for execution logs
CREATE INDEX IF NOT EXISTS idx_execution_logs_run_id ON execution_logs(run_id);
CREATE INDEX IF NOT EXISTS idx_execution_logs_timestamp ON execution_logs(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_execution_logs_level ON execution_logs(level);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for updated_at
DROP TRIGGER IF EXISTS update_plugin_metadata_updated_at ON plugin_metadata;
CREATE TRIGGER update_plugin_metadata_updated_at BEFORE UPDATE ON plugin_metadata
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_strategy_runs_updated_at ON strategy_runs;
CREATE TRIGGER update_strategy_runs_updated_at BEFORE UPDATE ON strategy_runs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
