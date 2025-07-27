package api

import (
	"net/http"
	"strconv"
	"time"
	"kanopt/internal/models"
	"kanopt/internal/messaging"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetTasks(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		boardID := c.Query("boardId")
		columnID := c.Query("columnId")
		
		var tasks []models.Task
		query := db.Preload("Assignee")
		
		if boardID != "" {
			if id, err := uuid.Parse(boardID); err == nil {
				query = query.Where("board_id = ?", id)
			}
		}
		
		if columnID != "" {
			if id, err := uuid.Parse(columnID); err == nil {
				query = query.Where("column_id = ?", id)
			}
		}
		
		result := query.Order("position ASC").Find(&tasks)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, tasks)
	}
}

func CreateTask(db *gorm.DB, rabbitmq *messaging.RabbitMQ) gin.HandlerFunc {
	return func(c *gin.Context) {
		var task models.Task
		
		if err := c.ShouldBindJSON(&task); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get the highest position in the column
		var maxPosition int
		db.Model(&models.Task{}).Where("column_id = ?", task.ColumnID).Select("COALESCE(MAX(position), -1)").Scan(&maxPosition)
		task.Position = maxPosition + 1

		result := db.Create(&task)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}

		// Load the task with relations
		db.Preload("Assignee").First(&task, task.ID)

		// Publish event
		event := messaging.Event{
			ID:      uuid.New().String(),
			Type:    "task.created",
			BoardID: task.BoardID.String(),
			UserID:  uuid.New().String(),
			Data: map[string]interface{}{
				"taskId":      task.ID,
				"columnId":    task.ColumnID,
				"title":       task.Title,
				"description": task.Description,
				"priority":    task.Priority,
				"storyPoints": task.StoryPoints,
				"position":    task.Position,
			},
		}
		
		if err := rabbitmq.PublishEvent(event); err != nil {
			c.Header("X-Event-Error", err.Error())
		}

		c.JSON(http.StatusCreated, task)
	}
}

func GetTask(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		taskID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
			return
		}

		var task models.Task
		result := db.Preload("Assignee").First(&task, taskID)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, task)
	}
}

func UpdateTask(db *gorm.DB, rabbitmq *messaging.RabbitMQ) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		taskID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
			return
		}

		var task models.Task
		if err := db.First(&task, taskID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var updateData models.Task
		if err := c.ShouldBindJSON(&updateData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Store old values for event
		oldStatus := "active"
		if task.CompletedAt != nil {
			oldStatus = "completed"
		}

		// Update fields
		task.Title = updateData.Title
		task.Description = updateData.Description
		task.Priority = updateData.Priority
		task.StoryPoints = updateData.StoryPoints
		task.Tags = updateData.Tags
		task.DueDate = updateData.DueDate
		task.AssigneeID = updateData.AssigneeID

		// Check if task is being marked as completed
		newStatus := "active"
		if updateData.CompletedAt != nil {
			task.CompletedAt = updateData.CompletedAt
			newStatus = "completed"
		}

		if err := db.Save(&task).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Load updated task with relations
		db.Preload("Assignee").First(&task, task.ID)

		// Publish event
		event := messaging.Event{
			ID:      uuid.New().String(),
			Type:    "task.updated",
			BoardID: task.BoardID.String(),
			UserID:  uuid.New().String(),
			Data: map[string]interface{}{
				"taskId":      task.ID,
				"title":       task.Title,
				"description": task.Description,
				"priority":    task.Priority,
				"storyPoints": task.StoryPoints,
				"oldStatus":   oldStatus,
				"status":      newStatus,
				"assigneeId":  task.AssigneeID,
			},
		}
		
		if err := rabbitmq.PublishEvent(event); err != nil {
			c.Header("X-Event-Error", err.Error())
		}

		c.JSON(http.StatusOK, task)
	}
}

func DeleteTask(db *gorm.DB, rabbitmq *messaging.RabbitMQ) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		taskID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
			return
		}

		var task models.Task
		if err := db.First(&task, taskID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if err := db.Delete(&task).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Publish event
		event := messaging.Event{
			ID:      uuid.New().String(),
			Type:    "task.deleted",
			BoardID: task.BoardID.String(),
			UserID:  uuid.New().String(),
			Data: map[string]interface{}{
				"taskId":   task.ID,
				"columnId": task.ColumnID,
				"title":    task.Title,
			},
		}
		
		if err := rabbitmq.PublishEvent(event); err != nil {
			c.Header("X-Event-Error", err.Error())
		}

		c.JSON(http.StatusOK, gin.H{"message": "Task deleted successfully"})
	}
}

func MoveTask(db *gorm.DB, rabbitmq *messaging.RabbitMQ) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		taskID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
			return
		}

		var moveData struct {
			ColumnID    uuid.UUID `json:"columnId" binding:"required"`
			Position    int       `json:"position" binding:"required"`
			BeforeTaskID *uuid.UUID `json:"beforeTaskId"`
			AfterTaskID  *uuid.UUID `json:"afterTaskId"`
		}

		if err := c.ShouldBindJSON(&moveData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var task models.Task
		if err := db.First(&task, taskID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		oldColumnID := task.ColumnID
		oldPosition := task.Position

		// Start transaction
		tx := db.Begin()

		// Update positions of other tasks in the old column
		if oldColumnID != moveData.ColumnID {
			err = tx.Model(&models.Task{}).
				Where("column_id = ? AND position > ?", oldColumnID, oldPosition).
				Update("position", gorm.Expr("position - 1")).Error
			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		// Update positions of tasks in the new column
		err = tx.Model(&models.Task{}).
			Where("column_id = ? AND position >= ?", moveData.ColumnID, moveData.Position).
			Update("position", gorm.Expr("position + 1")).Error
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Update the task
		task.ColumnID = moveData.ColumnID
		task.Position = moveData.Position

		if err := tx.Save(&task).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		tx.Commit()

		// Load updated task with relations
		db.Preload("Assignee").First(&task, task.ID)

		// Publish event
		event := messaging.Event{
			ID:      uuid.New().String(),
			Type:    "task.moved",
			BoardID: task.BoardID.String(),
			UserID:  uuid.New().String(),
			Data: map[string]interface{}{
				"taskId":       task.ID,
				"oldColumnId":  oldColumnID,
				"newColumnId":  moveData.ColumnID,
				"oldPosition":  oldPosition,
				"newPosition":  moveData.Position,
				"beforeTaskId": moveData.BeforeTaskID,
				"afterTaskId":  moveData.AfterTaskID,
			},
		}
		
		if err := rabbitmq.PublishEvent(event); err != nil {
			c.Header("X-Event-Error", err.Error())
		}

		c.JSON(http.StatusOK, task)
	}
}
