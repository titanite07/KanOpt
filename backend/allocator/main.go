package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Config struct {
	DatabaseURL string
	RabbitMQURL string
	APIBaseURL  string
	Port        string
}

type AllocatorAgent struct {
	db       *gorm.DB
	rabbitmq *amqp091.Connection
	channel  *amqp091.Channel
	logger   *logrus.Logger
	config   *Config
	mutex    sync.RWMutex
	
	// Agent state
	isActive        bool
	lastAnalysis    time.Time
	pendingActions  map[string]PendingAction
}

type PendingAction struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	BoardID     string                 `json:"boardId"`
	Priority    int                    `json:"priority"`
	Data        map[string]interface{} `json:"data"`
	CreatedAt   time.Time              `json:"createdAt"`
	RetryCount  int                    `json:"retryCount"`
}

type RiskAlert struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	BoardID   string                 `json:"boardId"`
	Level     string                 `json:"level"`
	Score     float64                `json:"score"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

type WorkloadAnalysis struct {
	UserID           string  `json:"userId"`
	Name             string  `json:"name"`
	ActiveTasks      int     `json:"activeTasks"`
	TotalStoryPoints int     `json:"totalStoryPoints"`
	AvgCycleTime     float64 `json:"avgCycleTime"`
	Capacity         float64 `json:"capacity"`
	IsOverloaded     bool    `json:"isOverloaded"`
}

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	config := &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://kanopt:kanopt@localhost:5432/kanopt?sslmode=disable"),
		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		APIBaseURL:  getEnv("API_BASE_URL", "http://localhost:8080"),
		Port:        getEnv("PORT", "8081"),
	}

	// Initialize database
	db, err := gorm.Open(postgres.Open(config.DatabaseURL), &gorm.Config{})
	if err != nil {
		logger.Fatal("Failed to connect to database:", err)
	}

	// Initialize RabbitMQ
	conn, err := amqp091.Dial(config.RabbitMQURL)
	if err != nil {
		logger.Fatal("Failed to connect to RabbitMQ:", err)
	}
	defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		logger.Fatal("Failed to open RabbitMQ channel:", err)
	}
	defer channel.Close()

	// Create allocator agent
	agent := &AllocatorAgent{
		db:             db,
		rabbitmq:       conn,
		channel:        channel,
		logger:         logger,
		config:         config,
		isActive:       true,
		pendingActions: make(map[string]PendingAction),
	}

	// Setup message queues
	if err := agent.setupQueues(); err != nil {
		logger.Fatal("Failed to setup queues:", err)
	}

	// Start background routines
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start event consumer
	go agent.consumeEvents(ctx)

	// Start periodic analysis
	go agent.runPeriodicAnalysis(ctx)

	// Start action executor
	go agent.executeActions(ctx)

	// Setup HTTP server for health checks and metrics
	router := gin.Default()
	setupRoutes(router, agent)

	server := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	// Start server
	go func() {
		logger.Infof("🤖 Allocator Agent starting on port %s", config.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server:", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("🛑 Shutting down Allocator Agent...")
	
	agent.isActive = false
	cancel()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("Server forced to shutdown:", err)
	}

	logger.Info("✅ Allocator Agent shutdown complete")
}

func (a *AllocatorAgent) setupQueues() error {
	// Declare exchange for risk alerts
	err := a.channel.ExchangeDeclare(
		"kanopt.risk",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Declare queue for risk alerts
	_, err = a.channel.QueueDeclare(
		"allocator.risk.queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Bind queue to exchange
	return a.channel.QueueBind(
		"allocator.risk.queue",
		"risk.*",
		"kanopt.risk",
		false,
		nil,
	)
}

func (a *AllocatorAgent) consumeEvents(ctx context.Context) {
	msgs, err := a.channel.Consume(
		"allocator.risk.queue",
		"allocator-agent",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		a.logger.Fatal("Failed to register consumer:", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case d := <-msgs:
			var alert RiskAlert
			if err := json.Unmarshal(d.Body, &alert); err != nil {
				a.logger.WithError(err).Error("Failed to unmarshal risk alert")
				d.Nack(false, false)
				continue
			}

			if err := a.handleRiskAlert(alert); err != nil {
				a.logger.WithError(err).Error("Failed to handle risk alert")
				d.Nack(false, true)
				continue
			}

			d.Ack(false)
		}
	}
}

func (a *AllocatorAgent) handleRiskAlert(alert RiskAlert) error {
	a.logger.WithFields(logrus.Fields{
		"alert_id":   alert.ID,
		"alert_type": alert.Type,
		"board_id":   alert.BoardID,
		"risk_level": alert.Level,
	}).Info("Processing risk alert")

	switch alert.Type {
	case "bottleneck":
		return a.handleBottleneckAlert(alert)
	case "overload":
		return a.handleOverloadAlert(alert)
	case "deadline_risk":
		return a.handleDeadlineAlert(alert)
	case "wip_violation":
		return a.handleWIPViolationAlert(alert)
	default:
		a.logger.WithField("alert_type", alert.Type).Warn("Unknown alert type")
		return nil
	}
}

func (a *AllocatorAgent) handleBottleneckAlert(alert RiskAlert) error {
	// Analyze column and suggest WIP limit adjustment or task redistribution
	columnID := alert.Data["columnId"].(string)
	
	action := PendingAction{
		ID:        uuid.New().String(),
		Type:      "adjust_wip_limits",
		BoardID:   alert.BoardID,
		Priority:  2,
		CreatedAt: time.Now(),
		Data: map[string]interface{}{
			"columnId":  columnID,
			"reason":    "bottleneck_detected",
			"alertId":   alert.ID,
			"newLimit":  int(alert.Data["wipLimit"].(float64)) + 2,
		},
	}

	return a.queueAction(action)
}

func (a *AllocatorAgent) handleOverloadAlert(alert RiskAlert) error {
	// Analyze workload and redistribute tasks
	userID := alert.Data["userId"].(string)
	
	// Get workload analysis
	analysis, err := a.analyzeWorkload(alert.BoardID)
	if err != nil {
		return err
	}

	// Find least loaded user
	var targetUser *WorkloadAnalysis
	for _, user := range analysis {
		if user.UserID != userID && !user.IsOverloaded {
			if targetUser == nil || user.ActiveTasks < targetUser.ActiveTasks {
				targetUser = &user
			}
		}
	}

	if targetUser == nil {
		a.logger.Warn("No available user for task redistribution")
		return nil
	}

	action := PendingAction{
		ID:        uuid.New().String(),
		Type:      "redistribute_tasks",
		BoardID:   alert.BoardID,
		Priority:  1,
		CreatedAt: time.Now(),
		Data: map[string]interface{}{
			"fromUserId": userID,
			"toUserId":   targetUser.UserID,
			"taskCount":  2,
			"reason":     "workload_balancing",
			"alertId":    alert.ID,
		},
	}

	return a.queueAction(action)
}

func (a *AllocatorAgent) handleDeadlineAlert(alert RiskAlert) error {
	// Prioritize task or suggest deadline extension
	taskID := alert.Data["taskId"].(string)
	
	action := PendingAction{
		ID:        uuid.New().String(),
		Type:      "reassign_overdue",
		BoardID:   alert.BoardID,
		Priority:  3,
		CreatedAt: time.Now(),
		Data: map[string]interface{}{
			"taskId":  taskID,
			"reason":  "deadline_risk",
			"alertId": alert.ID,
		},
	}

	return a.queueAction(action)
}

func (a *AllocatorAgent) handleWIPViolationAlert(alert RiskAlert) error {
	// Suggest WIP limit enforcement or task movement
	columnID := alert.Data["columnId"].(string)
	
	action := PendingAction{
		ID:        uuid.New().String(),
		Type:      "enforce_wip_limits",
		BoardID:   alert.BoardID,
		Priority:  2,
		CreatedAt: time.Now(),
		Data: map[string]interface{}{
			"columnId": columnID,
			"reason":   "wip_violation",
			"alertId":  alert.ID,
		},
	}

	return a.queueAction(action)
}

func (a *AllocatorAgent) queueAction(action PendingAction) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	
	a.pendingActions[action.ID] = action
	a.logger.WithFields(logrus.Fields{
		"action_id":   action.ID,
		"action_type": action.Type,
		"board_id":    action.BoardID,
		"priority":    action.Priority,
	}).Info("Action queued")
	
	return nil
}

func (a *AllocatorAgent) runPeriodicAnalysis(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !a.isActive {
				continue
			}

			a.logger.Info("Running periodic workload analysis")
			if err := a.performGlobalAnalysis(); err != nil {
				a.logger.WithError(err).Error("Periodic analysis failed")
			}
			
			a.lastAnalysis = time.Now()
		}
	}
}

func (a *AllocatorAgent) performGlobalAnalysis() error {
	// Get all active boards
	var boards []struct {
		ID string `json:"id"`
	}
	
	if err := a.db.Table("boards").Select("id").Find(&boards).Error; err != nil {
		return err
	}

	for _, board := range boards {
		if err := a.analyzeBoardHealth(board.ID); err != nil {
			a.logger.WithError(err).WithField("board_id", board.ID).Error("Board analysis failed")
		}
	}

	return nil
}

func (a *AllocatorAgent) analyzeBoardHealth(boardID string) error {
	// Analyze workload distribution
	analysis, err := a.analyzeWorkload(boardID)
	if err != nil {
		return err
	}

	// Check for imbalances
	var overloadedUsers, underutilizedUsers []WorkloadAnalysis
	for _, user := range analysis {
		if user.IsOverloaded {
			overloadedUsers = append(overloadedUsers, user)
		} else if user.ActiveTasks < 2 && user.Capacity > 0.3 {
			underutilizedUsers = append(underutilizedUsers, user)
		}
	}

	// Create rebalancing actions if needed
	if len(overloadedUsers) > 0 && len(underutilizedUsers) > 0 {
		for i, overloaded := range overloadedUsers {
			if i < len(underutilizedUsers) {
				action := PendingAction{
					ID:        uuid.New().String(),
					Type:      "redistribute_tasks",
					BoardID:   boardID,
					Priority:  2,
					CreatedAt: time.Now(),
					Data: map[string]interface{}{
						"fromUserId": overloaded.UserID,
						"toUserId":   underutilizedUsers[i].UserID,
						"taskCount":  2,
						"reason":     "proactive_balancing",
					},
				}
				a.queueAction(action)
			}
		}
	}

	return nil
}

func (a *AllocatorAgent) analyzeWorkload(boardID string) ([]WorkloadAnalysis, error) {
	// This would typically call the main API service
	url := fmt.Sprintf("%s/api/analytics/board/%s/team-performance", a.config.APIBaseURL, boardID)
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var teamPerformance []struct {
		UserID           string  `json:"userId"`
		Name             string  `json:"name"`
		CompletedTasks   int     `json:"completedTasks"`
		TotalStoryPoints int     `json:"totalStoryPoints"`
		AverageCycleTime float64 `json:"averageCycleTime"`
		Velocity         float64 `json:"velocity"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&teamPerformance); err != nil {
		return nil, err
	}

	// Convert to workload analysis
	var analysis []WorkloadAnalysis
	for _, perf := range teamPerformance {
		// Simple logic to determine if user is overloaded
		capacity := 1.0 - (float64(perf.CompletedTasks) / 10.0) // Assume max 10 tasks
		isOverloaded := perf.CompletedTasks > 6 || perf.TotalStoryPoints > 20

		analysis = append(analysis, WorkloadAnalysis{
			UserID:           perf.UserID,
			Name:             perf.Name,
			ActiveTasks:      perf.CompletedTasks, // Approximation
			TotalStoryPoints: perf.TotalStoryPoints,
			AvgCycleTime:     perf.AverageCycleTime,
			Capacity:         capacity,
			IsOverloaded:     isOverloaded,
		})
	}

	return analysis, nil
}

func (a *AllocatorAgent) executeActions(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !a.isActive {
				continue
			}

			a.executePendingActions()
		}
	}
}

func (a *AllocatorAgent) executePendingActions() {
	a.mutex.Lock()
	actions := make([]PendingAction, 0, len(a.pendingActions))
	for _, action := range a.pendingActions {
		actions = append(actions, action)
	}
	a.mutex.Unlock()

	// Sort by priority (higher priority first)
	for i := 0; i < len(actions)-1; i++ {
		for j := i + 1; j < len(actions); j++ {
			if actions[i].Priority < actions[j].Priority {
				actions[i], actions[j] = actions[j], actions[i]
			}
		}
	}

	for _, action := range actions {
		if err := a.executeAction(action); err != nil {
			a.logger.WithError(err).WithField("action_id", action.ID).Error("Action execution failed")
			
			// Retry logic
			action.RetryCount++
			if action.RetryCount < 3 {
				a.mutex.Lock()
				a.pendingActions[action.ID] = action
				a.mutex.Unlock()
			} else {
				a.logger.WithField("action_id", action.ID).Warn("Action failed after max retries")
				a.removeAction(action.ID)
			}
		} else {
			a.logger.WithField("action_id", action.ID).Info("Action executed successfully")
			a.removeAction(action.ID)
		}
	}
}

func (a *AllocatorAgent) executeAction(action PendingAction) error {
	url := fmt.Sprintf("%s/api/agent/actions", a.config.APIBaseURL)
	
	// Create agent action via API
	actionData := map[string]interface{}{
		"boardId":     action.BoardID,
		"type":        action.Type,
		"description": fmt.Sprintf("Autonomous action: %s", action.Type),
		"data":        action.Data,
	}

	jsonData, err := json.Marshal(actionData)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}

func (a *AllocatorAgent) removeAction(actionID string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	delete(a.pendingActions, actionID)
}

func setupRoutes(router *gin.Engine, agent *AllocatorAgent) {
	router.GET("/health", func(c *gin.Context) {
		status := "healthy"
		if !agent.isActive {
			status = "inactive"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":       status,
			"lastAnalysis": agent.lastAnalysis,
			"pendingActions": len(agent.pendingActions),
			"timestamp":    time.Now(),
		})
	})

	router.GET("/metrics", func(c *gin.Context) {
		agent.mutex.RLock()
		actions := make([]PendingAction, 0, len(agent.pendingActions))
		for _, action := range agent.pendingActions {
			actions = append(actions, action)
		}
		agent.mutex.RUnlock()

		c.JSON(http.StatusOK, gin.H{
			"isActive":       agent.isActive,
			"lastAnalysis":   agent.lastAnalysis,
			"pendingActions": actions,
			"totalActions":   len(actions),
		})
	})

	router.POST("/activate", func(c *gin.Context) {
		agent.isActive = true
		c.JSON(http.StatusOK, gin.H{"message": "Agent activated"})
	})

	router.POST("/deactivate", func(c *gin.Context) {
		agent.isActive = false
		c.JSON(http.StatusOK, gin.H{"message": "Agent deactivated"})
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
