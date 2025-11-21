package services

import (
	"context"

	"github.com/backtesting-org/kronos-sdk/pkg/types/plugin"
	"github.com/backtesting-org/live-trading/internal/database"
	"github.com/google/uuid"
)

// pluginStorageAdapter adapts database.Repository to plugin.Storage interface
type pluginStorageAdapter struct {
	repo *database.Repository
}

// NewPluginStorage creates a plugin storage adapter using the database repository
func newPluginStorageAdapter(repo *database.Repository) plugin.Storage {
	return &pluginStorageAdapter{repo: repo}
}

// SavePlugin implements plugin.Storage
func (s *pluginStorageAdapter) SavePlugin(ctx context.Context, metadata *plugin.Metadata) error {
	// Check if plugin exists
	existing, err := s.repo.GetPlugin(ctx, metadata.ID)
	if err == nil && existing != nil {
		// Update existing
		return s.repo.UpdatePlugin(ctx, s.toDBPlugin(metadata))
	}

	// Create new
	return s.repo.CreatePlugin(ctx, s.toDBPlugin(metadata))
}

// GetPlugin implements plugin.Storage
func (s *pluginStorageAdapter) GetPlugin(ctx context.Context, id uuid.UUID) (*plugin.Metadata, error) {
	dbPlugin, err := s.repo.GetPlugin(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toSDKMetadata(dbPlugin), nil
}

// ListPlugins implements plugin.Storage
func (s *pluginStorageAdapter) ListPlugins(ctx context.Context, limit, offset int) ([]*plugin.Metadata, error) {
	dbPlugins, err := s.repo.ListPlugins(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	result := make([]*plugin.Metadata, len(dbPlugins))
	for i, dbPlugin := range dbPlugins {
		result[i] = s.toSDKMetadata(&dbPlugin)
	}
	return result, nil
}

// DeletePlugin implements plugin.Storage
func (s *pluginStorageAdapter) DeletePlugin(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeletePlugin(ctx, id)
}

// toDBPlugin converts SDK metadata to database plugin
func (s *pluginStorageAdapter) toDBPlugin(metadata *plugin.Metadata) *database.PluginMetadata {
	dbPlugin := &database.PluginMetadata{
		ID:          metadata.ID,
		Name:        metadata.Name,
		Description: metadata.Description,
		PluginPath:  metadata.PluginPath,
		RiskLevel:   metadata.RiskLevel,
		Type:        metadata.Type,
		Version:     metadata.Version,
		CreatedBy:   metadata.CreatedBy,
	}

	// Convert parameters
	if metadata.Parameters != nil {
		dbPlugin.Parameters = make(database.ParameterDefMap)
		for name, param := range metadata.Parameters {
			dbPlugin.Parameters[name] = database.ParameterDef{
				Name:        param.Name,
				Type:        param.Type,
				Description: param.Description,
				Default:     param.Default,
				Required:    param.Required,
				Min:         param.Min,
				Max:         param.Max,
			}
		}
	}

	return dbPlugin
}

// toSDKMetadata converts database plugin to SDK metadata
func (s *pluginStorageAdapter) toSDKMetadata(dbPlugin *database.PluginMetadata) *plugin.Metadata {
	metadata := &plugin.Metadata{
		ID:          dbPlugin.ID,
		Name:        dbPlugin.Name,
		Description: dbPlugin.Description,
		PluginPath:  dbPlugin.PluginPath,
		RiskLevel:   dbPlugin.RiskLevel,
		Type:        dbPlugin.Type,
		Version:     dbPlugin.Version,
		CreatedBy:   dbPlugin.CreatedBy,
	}

	// Convert parameters
	if dbPlugin.Parameters != nil {
		metadata.Parameters = make(map[string]plugin.ParameterDef)
		for name, param := range dbPlugin.Parameters {
			metadata.Parameters[name] = plugin.ParameterDef{
				Name:        param.Name,
				Type:        param.Type,
				Description: param.Description,
				Default:     param.Default,
				Required:    param.Required,
				Min:         param.Min,
				Max:         param.Max,
			}
		}
	}

	return metadata
}
