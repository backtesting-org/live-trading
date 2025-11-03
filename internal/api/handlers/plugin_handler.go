package handlers

import (
	"io"
	"net/http"
	"strconv"

	"github.com/backtesting-org/live-trading/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PluginHandler handles plugin management endpoints
type PluginHandler struct {
	pluginManager *services.PluginManager
	logger        *zap.Logger
	maxUploadSize int64
}

// NewPluginHandler creates a new plugin handler
func NewPluginHandler(pluginManager *services.PluginManager, logger *zap.Logger, maxUploadSize int64) *PluginHandler {
	return &PluginHandler{
		pluginManager: pluginManager,
		logger:        logger,
		maxUploadSize: maxUploadSize,
	}
}

// UploadPlugin handles plugin file uploads
// POST /api/v1/plugins/upload
func (h *PluginHandler) UploadPlugin(c *gin.Context) {
	// Parse multipart form
	if err := c.Request.ParseMultipartForm(h.maxUploadSize); err != nil {
		h.logger.Error("Failed to parse multipart form", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid form data",
			Message: err.Error(),
		})
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("plugin")
	if err != nil {
		h.logger.Error("Failed to get uploaded file", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Missing plugin file",
			Message: "Please provide a plugin file in the 'plugin' field",
		})
		return
	}
	defer file.Close()

	// Validate file size
	if header.Size > h.maxUploadSize {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "File too large",
			Message: "Plugin file exceeds maximum allowed size",
		})
		return
	}

	// Read file content
	fileData, err := io.ReadAll(file)
	if err != nil {
		h.logger.Error("Failed to read file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to read file",
			Message: err.Error(),
		})
		return
	}

	// Get created_by from form or use default
	createdBy := c.PostForm("created_by")
	if createdBy == "" {
		createdBy = "system"
	}

	// Save plugin file
	pluginPath, err := h.pluginManager.SavePluginFile(header.Filename, fileData)
	if err != nil {
		h.logger.Error("Failed to save plugin file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to save plugin file",
			Message: err.Error(),
		})
		return
	}

	// Validate plugin
	if err := h.pluginManager.ValidatePluginFile(pluginPath); err != nil {
		h.logger.Error("Plugin validation failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid plugin file",
			Message: err.Error(),
		})
		return
	}

	// Load plugin and extract metadata
	metadata, err := h.pluginManager.LoadPlugin(c.Request.Context(), pluginPath, createdBy)
	if err != nil {
		h.logger.Error("Failed to load plugin", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to load plugin",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info("Plugin uploaded successfully",
		zap.String("id", metadata.ID.String()),
		zap.String("name", metadata.Name))

	c.JSON(http.StatusCreated, SuccessResponse{
		Message: "Plugin uploaded successfully",
		Data:    metadata,
	})
}

// ListPlugins lists all available plugins
// GET /api/v1/plugins
func (h *PluginHandler) ListPlugins(c *gin.Context) {
	// Parse pagination params
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 50
	}

	plugins, err := h.pluginManager.ListPlugins(c.Request.Context(), limit, offset)
	if err != nil {
		h.logger.Error("Failed to list plugins", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to list plugins",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Plugins retrieved successfully",
		Data: map[string]interface{}{
			"plugins": plugins,
			"count":   len(plugins),
			"limit":   limit,
			"offset":  offset,
		},
	})
}

// GetPlugin retrieves a specific plugin by ID
// GET /api/v1/plugins/:id
func (h *PluginHandler) GetPlugin(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid plugin ID",
			Message: "Plugin ID must be a valid UUID",
		})
		return
	}

	metadata, err := h.pluginManager.GetPluginMetadata(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get plugin", zap.Error(err))
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Plugin not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Plugin retrieved successfully",
		Data:    metadata,
	})
}

// DeletePlugin deletes a plugin
// DELETE /api/v1/plugins/:id
func (h *PluginHandler) DeletePlugin(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid plugin ID",
			Message: "Plugin ID must be a valid UUID",
		})
		return
	}

	if err := h.pluginManager.DeletePlugin(c.Request.Context(), id); err != nil {
		h.logger.Error("Failed to delete plugin", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to delete plugin",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info("Plugin deleted", zap.String("id", id.String()))

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Plugin deleted successfully",
		Data:    map[string]interface{}{"id": id.String()},
	})
}
