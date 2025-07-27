package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kanopt/internal/api"
	"kanopt/internal/config"
	"kanopt/internal/database"
	"kanopt/internal/messaging"
	"kanopt/internal/websocket"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Setup logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	if cfg.Environment == "development" {
		logger.SetLevel(logrus.DebugLevel)
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("Failed to connect to database:", err)
	}

	// Run migrations
	if err := database.Migrate(db); err != nil {
		logger.Fatal("Failed to run migrations:", err)
	}

	// Initialize RabbitMQ
	rabbitmq, err := messaging.NewRabbitMQ(cfg.RabbitMQURL, logger)
	if err != nil {
		logger.Fatal("Failed to connect to RabbitMQ:", err)
	}
	defer rabbitmq.Close()

	// Initialize WebSocket hub
	wsHub := websocket.NewHub(logger)
	go wsHub.Run()

	// Setup Gin router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS configuration
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"http://localhost:3000", "http://localhost:3001"}
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	router.Use(cors.New(corsConfig))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"timestamp": time.Now().UTC(),
			"services": gin.H{
				"database": "connected",
				"rabbitmq": "connected",
				"websocket": "active",
			},
			"stats": gin.H{
				"activeConnections": wsHub.GetConnectionCount(),
								"activeRooms": 0,
				"eventQueue":  0,
				"activeUsers": wsHub.GetConnectionCount(),
			},
		})
	})

	// API routes
	apiRoutes := router.Group("/api")
	{
		// Board management
		boards := apiRoutes.Group("/boards")
		{
			boards.GET("", api.GetBoards(db))
			boards.POST("", api.CreateBoard(db, rabbitmq))
			boards.GET("/:id", api.GetBoard(db))
			boards.PUT("/:id", api.UpdateBoard(db, rabbitmq))
			boards.DELETE("/:id", api.DeleteBoard(db, rabbitmq))
		}

		// Task management
		tasks := apiRoutes.Group("/tasks")
		{
			tasks.GET("", api.GetTasks(db))
			tasks.POST("", api.CreateTask(db, rabbitmq))
			tasks.GET("/:id", api.GetTask(db))
			tasks.PUT("/:id", api.UpdateTask(db, rabbitmq))
			tasks.DELETE("/:id", api.DeleteTask(db, rabbitmq))
			tasks.POST("/:id/move", api.MoveTask(db, rabbitmq))
		}

		// Analytics
		analytics := apiRoutes.Group("/analytics")
		{
			analytics.GET("/board/:id/velocity", api.GetVelocityMetrics(db))
			analytics.GET("/board/:id/burndown", api.GetBurndownData(db))
			analytics.GET("/board/:id/risk-trends", api.GetRiskTrends(db))
			analytics.GET("/board/:id/team-performance", api.GetTeamPerformance(db))
		}

		ai := apiRoutes.Group("/ai")
		{
			ai.GET("/board/:id/predictions", api.GetPredictions(db))
			ai.POST("/board/:id/risk-analysis", api.AnalyzeRisk(db))
		}

		// Agent actions
		agent := apiRoutes.Group("/agent")
		{
			agent.GET("/suggestions", api.GetSuggestions(db))
			agent.POST("/suggestions/:id/approve", api.ApproveSuggestion(db, rabbitmq))
			agent.POST("/suggestions/:id/reject", api.RejectSuggestion(db, rabbitmq))
			agent.GET("/actions", api.GetAgentActions(db))
			agent.POST("/actions/:id/execute", api.ExecuteAgentAction(db, rabbitmq))
		}

		// Events (event sourcing)
		events := apiRoutes.Group("/events")
		{
			events.GET("/board/:id", api.GetBoardEvents(db))
			events.POST("", api.CreateEvent(db, rabbitmq))
		}
	}

	// WebSocket endpoint
	router.GET("/ws", func(c *gin.Context) {
		websocket.HandleWebSocket(wsHub, c.Writer, c.Request, logger)
	})

	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logger.Infof("🚀 Starting Kanban API server on port %s", cfg.Port)
		logger.Infof("🌐 Environment: %s", cfg.Environment)
		logger.Infof("📊 Health check: http://localhost:%s/health", cfg.Port)
		logger.Infof("🔌 WebSocket: ws://localhost:%s/ws", cfg.Port)
		logger.Infof("📡 API docs: http://localhost:%s/api", cfg.Port)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server:", err)
		}
	}()

	// Start background event processor
	go func() {
		processor := messaging.NewEventProcessor(db, rabbitmq, logger)
		if err := processor.Start(); err != nil {
			logger.Error("Failed to start event processor:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("🛑 Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown:", err)
	}

	logger.Info("✅ Server shutdown complete")
}
