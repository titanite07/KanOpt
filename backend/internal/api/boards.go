package api

import (
	"net/http"
	"kanopt/internal/models"
	"kanopt/internal/messaging"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetBoards(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var boards []models.Board
		
		result := db.Preload("Columns").Preload("Tasks").Find(&boards)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, boards)
	}
}

func CreateBoard(db *gorm.DB, rabbitmq *messaging.RabbitMQ) gin.HandlerFunc {
	return func(c *gin.Context) {
		var board models.Board
		
		if err := c.ShouldBindJSON(&board); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Set created by from context (would come from JWT token in real app)
		board.CreatedBy = uuid.New()
		board.UpdatedBy = board.CreatedBy

		result := db.Create(&board)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}

		// Publish event
		event := messaging.Event{
			ID:        uuid.New().String(),
			Type:      "board.created",
			BoardID:   board.ID.String(),
			UserID:    board.CreatedBy.String(),
			Data: map[string]interface{}{
				"boardId":   board.ID,
				"name":      board.Name,
				"description": board.Description,
			},
		}
		
		if err := rabbitmq.PublishEvent(event); err != nil {
			// Log error but don't fail the request
			c.Header("X-Event-Error", err.Error())
		}

		c.JSON(http.StatusCreated, board)
	}
}

func GetBoard(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		boardID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID"})
			return
		}

		var board models.Board
		result := db.Preload("Columns").Preload("Tasks").Preload("Tasks.Assignee").First(&board, boardID)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Board not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, board)
	}
}

func UpdateBoard(db *gorm.DB, rabbitmq *messaging.RabbitMQ) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		boardID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID"})
			return
		}

		var board models.Board
		if err := db.First(&board, boardID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Board not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var updateData models.Board
		if err := c.ShouldBindJSON(&updateData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Update fields
		board.Name = updateData.Name
		board.Description = updateData.Description

		if err := db.Save(&board).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Publish event
		event := messaging.Event{
			ID:      uuid.New().String(),
			Type:    "board.updated",
			BoardID: board.ID.String(),
			UserID:  board.CreatedBy.String(),
			Data: map[string]interface{}{
				"boardId":     board.ID,
				"name":        board.Name,
				"description": board.Description,
			},
		}
		
		if err := rabbitmq.PublishEvent(event); err != nil {
			c.Header("X-Event-Error", err.Error())
		}

		c.JSON(http.StatusOK, board)
	}
}

func DeleteBoard(db *gorm.DB, rabbitmq *messaging.RabbitMQ) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		boardID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID"})
			return
		}

		var board models.Board
		if err := db.First(&board, boardID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Board not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Delete related records first
		if err := db.Where("board_id = ?", boardID).Delete(&models.Task{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if err := db.Where("board_id = ?", boardID).Delete(&models.Column{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Delete the board
		if err := db.Delete(&board).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Publish event
		event := messaging.Event{
			ID:      uuid.New().String(),
			Type:    "board.deleted",
			BoardID: board.ID.String(),
			UserID:  board.CreatedBy.String(),
			Data: map[string]interface{}{
				"boardId": board.ID,
			},
		}
		
		if err := rabbitmq.PublishEvent(event); err != nil {
			c.Header("X-Event-Error", err.Error())
		}

		c.JSON(http.StatusOK, gin.H{"message": "Board deleted successfully"})
	}
}
