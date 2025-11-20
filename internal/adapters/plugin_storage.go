package adapters

import (
	"context"

	plugintypes "github.com/backtesting-org/kronos-sdk/pkg/types/plugin"
	"github.com/backtesting-org/live-trading/internal/database"
	"github.com/google/uuid"
)

// PluginStorageAdapter adapts database.Repository to plugin.Storage interface
type PluginStorageAdapter struct {
	repo *database.Repository
}

// NewPluginStorageAdapter creates a new plugin storage adapter
func NewPluginStorageAdapter(repo *database.Repository) plugintypes.Storage {
	return &PluginStorageAdapter{repo: repo}
}

// SavePlugin stores plugin metadata
func (a *PluginStorageAdapter) SavePlugin(ctx context.Context, metadata *plugintypes.Metadata) error {
	dbMetadata := convertToDBMetadata(metadata)
	return a.repo.CreatePlugin(ctx, dbMetadata)
}

// GetPlugin retrieves plugin metadata by ID
func (a *PluginStorageAdapter) GetPlugin(ctx context.Context, id uuid.UUID) (*plugintypes.Metadata, error) {
	dbMetadata, err := a.repo.GetPlugin(ctx, id)
	if err != nil {
		return nil, err
	}
	return convertFromDBMetadata(dbMetadata), nil
}

// ListPlugins retrieves all plugins with pagination
func (a *PluginStorageAdapter) ListPlugins(ctx context.Context, limit, offset int) ([]*plugintypes.Metadata, error) {
	dbMetadatas, err := a.repo.ListPlugins(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	result := make([]*plugintypes.Metadata, len(dbMetadatas))
	for i, dbMeta := range dbMetadatas {
		result[i] = convertFromDBMetadata(&dbMeta)
	}
	return result, nil
}

// DeletePlugin soft deletes a plugin
func (a *PluginStorageAdapter) DeletePlugin(ctx context.Context, id uuid.UUID) error {
	return a.repo.DeletePlugin(ctx, id)
}

// convertToDBMetadata converts SDK metadata to database metadata
func convertToDBMetadata(m *plugintypes.Metadata) *database.PluginMetadata {
	params := make(database.ParameterDefMap)
	for k, v := range m.Parameters {
		params[k] = database.ParameterDef{
			Name:        v.Name,
			Type:        v.Type,
			Description: v.Description,
			Default:     v.Default,
			Required:    v.Required,
			Min:         v.Min,
			Max:         v.Max,
		}
	}

	return &database.PluginMetadata{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		RiskLevel:   m.RiskLevel,
		Type:        m.Type,
		Version:     m.Version,
		PluginPath:  m.PluginPath,
		CreatedBy:   m.CreatedBy,
		Parameters:  params,
	}
}

// convertFromDBMetadata converts database metadata to SDK metadata
func convertFromDBMetadata(m *database.PluginMetadata) *plugintypes.Metadata {
	params := make(map[string]plugintypes.ParameterDef)
	for k, v := range m.Parameters {
		params[k] = plugintypes.ParameterDef{
			Name:        v.Name,
			Type:        v.Type,
			Description: v.Description,
			Default:     v.Default,
			Required:    v.Required,
			Min:         v.Min,
			Max:         v.Max,
		}
	}

	return &plugintypes.Metadata{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		RiskLevel:   m.RiskLevel,
		Type:        m.Type,
		Version:     m.Version,
		PluginPath:  m.PluginPath,
		CreatedBy:   m.CreatedBy,
		Parameters:  params,
	}
}
