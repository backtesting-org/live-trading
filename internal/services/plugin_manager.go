package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"

	kronosTypes "github.com/backtesting-org/kronos-sdk/pkg/types/kronos"
	"github.com/backtesting-org/kronos-sdk/pkg/types/strategy"
	"github.com/backtesting-org/live-trading/internal/config"
	"github.com/backtesting-org/live-trading/internal/database"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PluginManager handles plugin loading and management
type PluginManager struct {
	repo          *database.Repository
	logger        *zap.Logger
	pluginDir     string
	loadedPlugins map[uuid.UUID]*LoadedPlugin
	mu            sync.RWMutex
}

// LoadedPlugin represents a plugin that has been loaded into memory
type LoadedPlugin struct {
	ID           uuid.UUID
	Name         string
	Plugin       *plugin.Plugin
	StrategyFunc func() strategy.Strategy
	Metadata     *database.PluginMetadata
}

// KronosAware is implemented by strategies that accept a Kronos context injection
type KronosAware interface {
    SetKronos(kronosTypes.Kronos)
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(repo *database.Repository, logger *zap.Logger, cfg *config.Config) *PluginManager {
	return &PluginManager{
		repo:          repo,
		logger:        logger,
		pluginDir:     cfg.Plugin.Directory,
		loadedPlugins: make(map[uuid.UUID]*LoadedPlugin),
	}
}

// LoadPlugin loads a plugin from a file path and stores its metadata
func (pm *PluginManager) LoadPlugin(ctx context.Context, pluginPath, createdBy string) (*database.PluginMetadata, error) {
	pm.logger.Info("Loading plugin", zap.String("path", pluginPath))

	// Validate file exists
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin file does not exist: %s", pluginPath)
	}

	// Load the plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %w", err)
	}

	// Look up the NewStrategy symbol
	newStrategySymbol, err := p.Lookup("NewStrategy")
	if err != nil {
		// Try alternative symbol: Strategy variable export
		strategySymbol, err := p.Lookup("Strategy")
		if err != nil {
			return nil, fmt.Errorf("plugin must export NewStrategy function or Strategy variable: %w", err)
		}

		// Type assert to strategy.Strategy
		strat, ok := strategySymbol.(*strategy.Strategy)
		if !ok || strat == nil {
			return nil, fmt.Errorf("Strategy symbol is not of type strategy.Strategy")
		}

		// Extract metadata from strategy instance
		metadata, err := pm.extractMetadataFromStrategy(*strat)
		if err != nil {
			return nil, fmt.Errorf("failed to extract metadata: %w", err)
		}

		// Generate unique ID
		metadata.ID = uuid.New()
		metadata.PluginPath = pluginPath
		metadata.CreatedBy = createdBy

		// Store in database
		if err := pm.repo.CreatePlugin(ctx, metadata); err != nil {
			return nil, fmt.Errorf("failed to store plugin metadata: %w", err)
		}

		pm.logger.Info("Plugin loaded successfully",
			zap.String("id", metadata.ID.String()),
			zap.String("name", metadata.Name))

		return metadata, nil
	}

	// Type assert NewStrategy function
	newStrategyFunc, ok := newStrategySymbol.(func() strategy.Strategy)
	if !ok {
		return nil, fmt.Errorf("NewStrategy must be a function returning strategy.Strategy")
	}

	// Create a temporary instance to extract metadata
	tempStrategy := newStrategyFunc()
	if tempStrategy == nil {
		return nil, fmt.Errorf("NewStrategy() returned nil")
	}

	// Extract metadata
	metadata, err := pm.extractMetadataFromStrategy(tempStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata: %w", err)
	}

	// Generate unique ID
	metadata.ID = uuid.New()
	metadata.PluginPath = pluginPath
	metadata.CreatedBy = createdBy

	// Store in database
	if err := pm.repo.CreatePlugin(ctx, metadata); err != nil {
		return nil, fmt.Errorf("failed to store plugin metadata: %w", err)
	}

	// Cache the loaded plugin
	pm.mu.Lock()
	pm.loadedPlugins[metadata.ID] = &LoadedPlugin{
		ID:           metadata.ID,
		Name:         metadata.Name,
		Plugin:       p,
		StrategyFunc: newStrategyFunc,
		Metadata:     metadata,
	}
	pm.mu.Unlock()

	pm.logger.Info("Plugin loaded successfully",
		zap.String("id", metadata.ID.String()),
		zap.String("name", metadata.Name))

	return metadata, nil
}

// extractMetadataFromStrategy extracts metadata from a strategy instance
func (pm *PluginManager) extractMetadataFromStrategy(strat strategy.Strategy) (*database.PluginMetadata, error) {
	metadata := &database.PluginMetadata{
		Name:        string(strat.GetName()),
		Description: strat.GetDescription(),
		RiskLevel:   string(strat.GetRiskLevel()),
		Type:        string(strat.GetStrategyType()),
		Version:     "1.0.0", // Default version
		Parameters:  make(database.ParameterDefMap),
	}

	// Try to extract parameters if strategy implements ParameterProvider interface
	if paramProvider, ok := strat.(ParameterProvider); ok {
		params := paramProvider.GetParameters()
		for _, p := range params {
			metadata.Parameters[p.Name] = database.ParameterDef{
				Name:        p.Name,
				Type:        p.Type,
				Description: p.Description,
				Default:     p.Default,
				Required:    p.Required,
				Min:         p.Min,
				Max:         p.Max,
			}
		}
	}

	return metadata, nil
}

// GetLoadedPlugin retrieves a loaded plugin by ID
func (pm *PluginManager) GetLoadedPlugin(ctx context.Context, id uuid.UUID) (*LoadedPlugin, error) {
	pm.mu.RLock()
	loaded, exists := pm.loadedPlugins[id]
	pm.mu.RUnlock()

	if exists {
		return loaded, nil
	}

	// Plugin not in memory, try to load from database and file
	metadata, err := pm.repo.GetPlugin(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("plugin not found in database: %w", err)
	}

	// Load the plugin file
	p, err := plugin.Open(metadata.PluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin file: %w", err)
	}

	// Try to get NewStrategy function
	newStrategySymbol, err := p.Lookup("NewStrategy")
	if err != nil {
		return nil, fmt.Errorf("plugin must export NewStrategy function: %w", err)
	}

	newStrategyFunc, ok := newStrategySymbol.(func() strategy.Strategy)
	if !ok {
		return nil, fmt.Errorf("NewStrategy must be a function returning strategy.Strategy")
	}

	// Cache and return
	loaded = &LoadedPlugin{
		ID:           metadata.ID,
		Name:         metadata.Name,
		Plugin:       p,
		StrategyFunc: newStrategyFunc,
		Metadata:     metadata,
	}

	pm.mu.Lock()
	pm.loadedPlugins[id] = loaded
	pm.mu.Unlock()

	return loaded, nil
}

// InstantiateStrategy creates a new strategy instance from a loaded plugin
func (pm *PluginManager) InstantiateStrategy(ctx context.Context, id uuid.UUID) (strategy.Strategy, error) {
	loaded, err := pm.GetLoadedPlugin(ctx, id)
	if err != nil {
		return nil, err
	}

	if loaded.StrategyFunc == nil {
		return nil, fmt.Errorf("plugin does not have a valid NewStrategy function")
	}

	strat := loaded.StrategyFunc()
	if strat == nil {
		return nil, fmt.Errorf("NewStrategy() returned nil")
	}

	return strat, nil
}

// ListPlugins retrieves all plugins from the database
func (pm *PluginManager) ListPlugins(ctx context.Context, limit, offset int) ([]database.PluginMetadata, error) {
	return pm.repo.ListPlugins(ctx, limit, offset)
}

// GetPluginMetadata retrieves plugin metadata by ID
func (pm *PluginManager) GetPluginMetadata(ctx context.Context, id uuid.UUID) (*database.PluginMetadata, error) {
	return pm.repo.GetPlugin(ctx, id)
}

// DeletePlugin removes a plugin
func (pm *PluginManager) DeletePlugin(ctx context.Context, id uuid.UUID) error {
	// Remove from cache
	pm.mu.Lock()
	delete(pm.loadedPlugins, id)
	pm.mu.Unlock()

	// Soft delete from database
	if err := pm.repo.DeletePlugin(ctx, id); err != nil {
		return err
	}

	pm.logger.Info("Plugin deleted", zap.String("id", id.String()))
	return nil
}

// SavePluginFile saves an uploaded plugin file to the plugin directory
func (pm *PluginManager) SavePluginFile(fileName string, data []byte) (string, error) {
	// Ensure plugin directory exists
	if err := os.MkdirAll(pm.pluginDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// Generate unique filename
	pluginID := uuid.New()
	ext := filepath.Ext(fileName)
	if ext != ".so" {
		return "", fmt.Errorf("invalid plugin file extension: must be .so")
	}

	uniqueFileName := fmt.Sprintf("%s_%s%s", pluginID.String(), filepath.Base(fileName[:len(fileName)-len(ext)]), ext)
	filePath := filepath.Join(pm.pluginDir, uniqueFileName)

	// Write file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write plugin file: %w", err)
	}

	pm.logger.Info("Plugin file saved", zap.String("path", filePath))
	return filePath, nil
}

// ValidatePluginFile validates that a file is a valid Go plugin
func (pm *PluginManager) ValidatePluginFile(filePath string) error {
	// Try to open the plugin
	p, err := plugin.Open(filePath)
	if err != nil {
		return fmt.Errorf("invalid plugin file: %w", err)
	}

	// Check for required symbols
	_, err = p.Lookup("NewStrategy")
	if err != nil {
		// Try alternative symbol
		_, err = p.Lookup("Strategy")
		if err != nil {
			return fmt.Errorf("plugin must export NewStrategy function or Strategy variable")
		}
	}

	return nil
}

// ParameterProvider is an optional interface that strategies can implement
// to provide parameter definitions
type ParameterProvider interface {
	GetParameters() []ParameterDef
}

// ParameterDef defines a strategy parameter
type ParameterDef struct {
	Name        string
	Type        string
	Description string
	Default     interface{}
	Required    bool
	Min         interface{}
	Max         interface{}
}
