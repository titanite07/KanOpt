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

func GetBoardEvents(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		boardID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID"})
			return
		}

		// Query parameters
		eventType := c.Query("type")
		limit := c.DefaultQuery("limit", "100")
		offset := c.DefaultQuery("offset", "0")
		since := c.Query("since")

		query := db.Where("board_id = ?", boardID).Preload("User")

		if eventType != "" {
			query = query.Where("type = ?", eventType)
		}

		if since != "" {
			if sinceTime, err := time.Parse(time.RFC3339, since); err == nil {
				query = query.Where("timestamp >= ?", sinceTime)
			}
		}

		var events []models.Event
		result := query.Order("timestamp DESC").
			Limit(parseLimit(limit)).
			Offset(parseOffset(offset)).
			Find(&events)

		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}

		// Get total count for pagination
		var total int64
		countQuery := db.Model(&models.Event{}).Where("board_id = ?", boardID)
		if eventType != "" {
			countQuery = countQuery.Where("type = ?", eventType)
		}
		if since != "" {
			if sinceTime, err := time.Parse(time.RFC3339, since); err == nil {
				countQuery = countQuery.Where("timestamp >= ?", sinceTime)
			}
		}
		countQuery.Count(&total)

		c.JSON(http.StatusOK, gin.H{
			"events": events,
			"pagination": gin.H{
				"total":  total,
				"limit":  parseLimit(limit),
				"offset": parseOffset(offset),
			},
		})
	}
}

func CreateEvent(db *gorm.DB, rabbitmq *messaging.RabbitMQ) gin.HandlerFunc {
	return func(c *gin.Context) {
		var eventData struct {
			Type    string                 `json:"type" binding:"required"`
			BoardID uuid.UUID              `json:"boardId" binding:"required"`
			Data    map[string]interface{} `json:"data"`
		}

		if err := c.ShouldBindJSON(&eventData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Create event
		event := models.Event{
			BoardID:   eventData.BoardID,
			Type:      eventData.Type,
			Data:      eventData.Data,
			UserID:    uuid.New(),
			Timestamp: time.Now(),
		}

		if err := db.Create(&event).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Publish to RabbitMQ
		rabbitEvent := messaging.Event{
			ID:        event.ID.String(),
			Type:      event.Type,
			BoardID:   event.BoardID.String(),
			UserID:    event.UserID.String(),
			Data:      event.Data,
			Timestamp: event.Timestamp,
		}

		if err := rabbitmq.PublishEvent(rabbitEvent); err != nil {
			c.Header("X-Event-Error", err.Error())
		}

		// Load event with user data
		db.Preload("User").First(&event, event.ID)

		c.JSON(http.StatusCreated, event)
	}
}

// Helper functions for pagination
func parseLimit(limitStr string) int {
	if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
		return limit
	}
	return 100 // Default limit
}

func parseOffset(offsetStr string) int {
	if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
		return offset
	}
	return 0 // Default offset
}
