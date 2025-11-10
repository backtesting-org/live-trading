package api

import (
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/temporal"
	"github.com/backtesting-org/live-trading/internal/api/handlers"
	"github.com/backtesting-org/live-trading/internal/api/websocket"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SetupRouter sets up the API router
func SetupRouter(
	pluginHandler *handlers.PluginHandler,
	strategyHandler *handlers.StrategyHandler,
	ordersHandler *handlers.OrdersHandler,
	wsHandler *websocket.Handler,
	logger *zap.Logger,
	corsAllowOrigin string,
	timeProvider temporal.TimeProvider,
) *gin.Engine {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Middleware
	router.Use(gin.Recovery())
	router.Use(LoggerMiddleware(logger, timeProvider))

	// CORS configuration
	config := cors.Config{
		AllowOrigins:     []string{corsAllowOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(config))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "live-trading-api",
			"version": "1.0.0",
		})
	})

	// WebSocket endpoint
	router.GET("/ws", wsHandler.HandleConnection)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Plugin management
		plugins := v1.Group("/plugins")
		{
			plugins.POST("/upload", pluginHandler.UploadPlugin)
			plugins.GET("", pluginHandler.ListPlugins)
			plugins.GET("/:id", pluginHandler.GetPlugin)
			plugins.DELETE("/:id", pluginHandler.DeletePlugin)
		}

		// Strategy execution
		strategies := v1.Group("/strategies")
		{
			strategies.POST("/start", strategyHandler.StartStrategy)
			strategies.GET("/runs", strategyHandler.ListRuns)
			strategies.POST("/:runId/stop", strategyHandler.StopStrategy)
			strategies.GET("/:runId/status", strategyHandler.GetRunStatus)
			strategies.GET("/:runId/stats", strategyHandler.GetRunStats)
		}

		// Order management
		orders := v1.Group("/orders")
		{
			orders.GET("/open", ordersHandler.GetOpenOrders)
			orders.GET("/:order_id", ordersHandler.GetOrder)
		}

		// Account information
		account := v1.Group("/account")
		{
			account.GET("/summary", ordersHandler.GetAccountSummary)
			account.GET("/balances", ordersHandler.GetBalances)
			account.GET("/positions", ordersHandler.GetPositions)
			account.GET("/trades", ordersHandler.GetTradeHistory)
			account.GET("/sub-accounts", ordersHandler.GetSubAccounts)
			account.GET("/info", ordersHandler.GetAccountInfo)
		}
	}

	return router
}

// LoggerMiddleware creates a Gin middleware for logging
func LoggerMiddleware(logger *zap.Logger, timeProvider temporal.TimeProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := timeProvider.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		end := timeProvider.Now()
		latency := end.Sub(start)

		if len(c.Errors) > 0 {
			// Log errors if any
			for _, e := range c.Errors.Errors() {
				logger.Error("Request error", zap.String("error", e))
			}
		} else {
			logger.Info("Request",
				zap.Int("status", c.Writer.Status()),
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.String("query", query),
				zap.String("ip", c.ClientIP()),
				zap.Duration("latency", latency),
			)
		}
	}
}
