package messaging

import (
	"encoding/json"
	"fmt"
	"kanopt/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type EventProcessor struct {
	db       *gorm.DB
	rabbitmq *RabbitMQ
	logger   *logrus.Logger
}

func NewEventProcessor(db *gorm.DB, rabbitmq *RabbitMQ, logger *logrus.Logger) *EventProcessor {
	return &EventProcessor{
		db:       db,
		rabbitmq: rabbitmq,
		logger:   logger,
	}
}

func (ep *EventProcessor) Start() error {
	return ep.rabbitmq.ConsumeEvents(ep.handleEvent)
}

func (ep *EventProcessor) handleEvent(event Event) error {
	ep.logger.WithFields(logrus.Fields{
		"event_id":   event.ID,
		"event_type": event.Type,
		"board_id":   event.BoardID,
	}).Info("Processing event")

	// Store event in database
	if err := ep.storeEvent(event); err != nil {
		return fmt.Errorf("failed to store event: %w", err)
	}

	// Process different event types
	switch event.Type {
	case "task.created":
		return ep.handleTaskCreated(event)
	case "task.updated":
		return ep.handleTaskUpdated(event)
	case "task.moved":
		return ep.handleTaskMoved(event)
	case "task.deleted":
		return ep.handleTaskDeleted(event)
	case "board.created":
		return ep.handleBoardCreated(event)
	case "board.updated":
		return ep.handleBoardUpdated(event)
	case "column.created":
		return ep.handleColumnCreated(event)
	case "column.updated":
		return ep.handleColumnUpdated(event)
	default:
		ep.logger.WithField("event_type", event.Type).Warn("Unknown event type")
		return nil
	}
}

func (ep *EventProcessor) storeEvent(event Event) error {
	boardID, err := uuid.Parse(event.BoardID)
	if err != nil {
		return fmt.Errorf("invalid board ID: %w", err)
	}

	userID, err := uuid.Parse(event.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	dbEvent := models.Event{
		BoardID:   boardID,
		Type:      event.Type,
		Data:      event.Data,
		UserID:    userID,
		Timestamp: event.Timestamp,
	}

	return ep.db.Create(&dbEvent).Error
}

func (ep *EventProcessor) handleTaskCreated(event Event) error {
	// Update velocity metrics
	return ep.updateVelocityMetrics(event.BoardID)
}

func (ep *EventProcessor) handleTaskUpdated(event Event) error {
	// Check if task was completed
	if status, ok := event.Data["status"].(string); ok && status == "completed" {
		// Update completion metrics
		return ep.updateCompletionMetrics(event.BoardID)
	}
	return nil
}

func (ep *EventProcessor) handleTaskMoved(event Event) error {
	// Update cycle time metrics
	if err := ep.updateCycleTimeMetrics(event.BoardID); err != nil {
		return err
	}

	// Check for bottlenecks
	return ep.analyzeBottlenecks(event.BoardID)
}

func (ep *EventProcessor) handleTaskDeleted(event Event) error {
	// Update velocity metrics
	return ep.updateVelocityMetrics(event.BoardID)
}

func (ep *EventProcessor) handleBoardCreated(event Event) error {
	// Initialize default columns if not exists
	return ep.initializeDefaultColumns(event.BoardID)
}

func (ep *EventProcessor) handleBoardUpdated(event Event) error {
	// No specific processing needed for board updates
	return nil
}

func (ep *EventProcessor) handleColumnCreated(event Event) error {
	// No specific processing needed for column creation
	return nil
}

func (ep *EventProcessor) handleColumnUpdated(event Event) error {
	// Check WIP limit violations
	return ep.checkWIPLimits(event.BoardID)
}

func (ep *EventProcessor) updateVelocityMetrics(boardIDStr string) error {
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		return err
	}

	// Calculate current week velocity
	now := time.Now()
	weekStart := now.AddDate(0, 0, -int(now.Weekday()))
	
	var completedTasks []models.Task
	err = ep.db.Where("board_id = ? AND completed_at >= ?", boardID, weekStart).Find(&completedTasks).Error
	if err != nil {
		return err
	}

	totalPoints := 0
	for _, task := range completedTasks {
		totalPoints += task.StoryPoints
	}

	velocity := float64(totalPoints) / float64(len(completedTasks)+1) // Avoid division by zero

	// Get current week number
	_, week := now.ISOWeek()

	// Update or create velocity metric
	velocityMetric := models.VelocityMetric{
		BoardID:     boardID,
		SprintWeek:  week,
		Velocity:    velocity,
		Completed:   len(completedTasks),
		TotalPoints: totalPoints,
		Throughput:  len(completedTasks),
	}

	return ep.db.Save(&velocityMetric).Error
}

func (ep *EventProcessor) updateCompletionMetrics(boardIDStr string) error {
	// Similar to velocity metrics but focused on completion rates
	return ep.updateVelocityMetrics(boardIDStr)
}

func (ep *EventProcessor) updateCycleTimeMetrics(boardIDStr string) error {
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		return err
	}

	// Calculate average cycle time for completed tasks
	var tasks []models.Task
	err = ep.db.Where("board_id = ? AND completed_at IS NOT NULL", boardID).Find(&tasks).Error
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		return nil
	}

	totalCycleTime := float64(0)
	for _, task := range tasks {
		if task.CompletedAt != nil {
			cycleTime := task.CompletedAt.Sub(task.CreatedAt).Hours() / 24 // Days
			totalCycleTime += cycleTime
		}
	}

	avgCycleTime := totalCycleTime / float64(len(tasks))

	// Update velocity metric with cycle time
	now := time.Now()
	_, week := now.ISOWeek()

	var velocityMetric models.VelocityMetric
	err = ep.db.Where("board_id = ? AND sprint_week = ?", boardID, week).First(&velocityMetric).Error
	if err != nil {
		// Create new if not exists
		velocityMetric = models.VelocityMetric{
			BoardID:    boardID,
			SprintWeek: week,
			CycleTime:  avgCycleTime,
		}
	} else {
		velocityMetric.CycleTime = avgCycleTime
	}

	return ep.db.Save(&velocityMetric).Error
}

func (ep *EventProcessor) analyzeBottlenecks(boardIDStr string) error {
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		return err
	}

	// Count tasks in each column
	var columns []models.Column
	err = ep.db.Where("board_id = ?", boardID).Preload("Tasks").Find(&columns).Error
	if err != nil {
		return err
	}

	for _, column := range columns {
		if column.WIPLimit > 0 && len(column.Tasks) > column.WIPLimit {
			// Create risk prediction for bottleneck
			risk := models.RiskPrediction{
				BoardID:     boardID,
				Type:        "bottleneck",
				Level:       "high",
				Score:       0.8,
				Description: fmt.Sprintf("Column '%s' exceeds WIP limit: %d/%d tasks", column.Name, len(column.Tasks), column.WIPLimit),
				Data: map[string]interface{}{
					"columnId":    column.ID,
					"columnName":  column.Name,
					"taskCount":   len(column.Tasks),
					"wipLimit":    column.WIPLimit,
					"overflowBy":  len(column.Tasks) - column.WIPLimit,
				},
			}

			if err := ep.db.Create(&risk).Error; err != nil {
				ep.logger.WithError(err).Error("Failed to create risk prediction")
			}
		}
	}

	return nil
}

func (ep *EventProcessor) checkWIPLimits(boardIDStr string) error {
	// Similar to bottleneck analysis
	return ep.analyzeBottlenecks(boardIDStr)
}

func (ep *EventProcessor) initializeDefaultColumns(boardIDStr string) error {
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		return err
	}

	// Check if columns already exist
	var count int64
	ep.db.Model(&models.Column{}).Where("board_id = ?", boardID).Count(&count)
	if count > 0 {
		return nil // Columns already exist
	}

	// Create default columns
	defaultColumns := []models.Column{
		{BoardID: boardID, Name: "Backlog", Position: 0, WIPLimit: 0},
		{BoardID: boardID, Name: "To Do", Position: 1, WIPLimit: 5},
		{BoardID: boardID, Name: "In Progress", Position: 2, WIPLimit: 3},
		{BoardID: boardID, Name: "Review", Position: 3, WIPLimit: 2},
		{BoardID: boardID, Name: "Done", Position: 4, WIPLimit: 0},
	}

	return ep.db.Create(&defaultColumns).Error
}
