package api

import (
	"fmt"
	"net/http"
	"time"
	"kanopt/internal/models"
	"kanopt/internal/messaging"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetSuggestions(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		boardID := c.Query("boardId")
		status := c.Query("status")
		
		query := db.Model(&models.Suggestion{})
		
		if boardID != "" {
			if id, err := uuid.Parse(boardID); err == nil {
				query = query.Where("board_id = ?", id)
			}
		}
		
		if status != "" {
			query = query.Where("status = ?", status)
		} else {
			query = query.Where("status = ?", "pending")
		}
		
		var suggestions []models.Suggestion
		result := query.Order("priority DESC, created_at DESC").Find(&suggestions)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, suggestions)
	}
}

func ApproveSuggestion(db *gorm.DB, rabbitmq *messaging.RabbitMQ) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		suggestionID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid suggestion ID"})
			return
		}

		var suggestion models.Suggestion
		if err := db.First(&suggestion, suggestionID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Suggestion not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if suggestion.Status != "pending" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Suggestion already processed"})
			return
		}

		// Update suggestion status
		suggestion.Status = "approved"
		if err := db.Save(&suggestion).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Create agent action to execute the suggestion
		agentAction := models.AgentAction{
			BoardID:     suggestion.BoardID,
			Type:        suggestion.Type,
			Description: "Executing approved suggestion: " + suggestion.Title,
			Data:        suggestion.Data,
			Status:      "pending",
		}

		if err := db.Create(&agentAction).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Publish event
		event := messaging.Event{
			ID:      uuid.New().String(),
			Type:    "suggestion.approved",
			BoardID: suggestion.BoardID.String(),
			UserID:  uuid.New().String(),
			Data: map[string]interface{}{
				"suggestionId": suggestion.ID,
				"actionId":     agentAction.ID,
				"type":         suggestion.Type,
				"title":        suggestion.Title,
			},
		}
		
		if err := rabbitmq.PublishEvent(event); err != nil {
			c.Header("X-Event-Error", err.Error())
		}

		c.JSON(http.StatusOK, gin.H{
			"message":      "Suggestion approved",
			"suggestion":   suggestion,
			"agentAction":  agentAction,
		})
	}
}

func RejectSuggestion(db *gorm.DB, rabbitmq *messaging.RabbitMQ) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		suggestionID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid suggestion ID"})
			return
		}

		var suggestion models.Suggestion
		if err := db.First(&suggestion, suggestionID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Suggestion not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if suggestion.Status != "pending" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Suggestion already processed"})
			return
		}

		// Update suggestion status
		suggestion.Status = "rejected"
		if err := db.Save(&suggestion).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Publish event
		event := messaging.Event{
			ID:      uuid.New().String(),
			Type:    "suggestion.rejected",
			BoardID: suggestion.BoardID.String(),
			UserID:  uuid.New().String(),
			Data: map[string]interface{}{
				"suggestionId": suggestion.ID,
				"type":         suggestion.Type,
				"title":        suggestion.Title,
			},
		}
		
		if err := rabbitmq.PublishEvent(event); err != nil {
			c.Header("X-Event-Error", err.Error())
		}

		c.JSON(http.StatusOK, gin.H{
			"message":    "Suggestion rejected",
			"suggestion": suggestion,
		})
	}
}

func GetAgentActions(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		boardID := c.Query("boardId")
		status := c.Query("status")
		
		query := db.Model(&models.AgentAction{})
		
		if boardID != "" {
			if id, err := uuid.Parse(boardID); err == nil {
				query = query.Where("board_id = ?", id)
			}
		}
		
		if status != "" {
			query = query.Where("status = ?", status)
		}
		
		var actions []models.AgentAction
		result := query.Order("created_at DESC").Find(&actions)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, actions)
	}
}

func ExecuteAgentAction(db *gorm.DB, rabbitmq *messaging.RabbitMQ) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		actionID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action ID"})
			return
		}

		var action models.AgentAction
		if err := db.First(&action, actionID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Agent action not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if action.Status != "pending" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Action already processed"})
			return
		}

		// Execute the action based on type
		var executionResult map[string]interface{}
		var executionError error

		switch action.Type {
		case "redistribute_tasks":
			executionResult, executionError = executeTaskRedistribution(db, action)
		case "adjust_wip_limits":
			executionResult, executionError = executeWIPAdjustment(db, action)
		case "create_subtasks":
			executionResult, executionError = executeSubtaskCreation(db, action)
		case "reassign_overdue":
			executionResult, executionError = executeOverdueReassignment(db, action)
		default:
			executionError = fmt.Errorf("unknown action type: %s", action.Type)
		}

		// Update action status
		now := time.Now()
		if executionError != nil {
			action.Status = "failed"
			action.Data["error"] = executionError.Error()
		} else {
			action.Status = "completed"
			action.Data["result"] = executionResult
			action.ExecutedAt = &now
		}

		if err := db.Save(&action).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Publish event
		event := messaging.Event{
			ID:      uuid.New().String(),
			Type:    "agent.action.executed",
			BoardID: action.BoardID.String(),
			UserID:  uuid.New().String(),
			Data: map[string]interface{}{
				"actionId":    action.ID,
				"actionType":  action.Type,
				"status":      action.Status,
				"result":      executionResult,
			},
		}
		
		if err := rabbitmq.PublishEvent(event); err != nil {
			c.Header("X-Event-Error", err.Error())
		}

		response := gin.H{
			"message": "Agent action executed",
			"action":  action,
		}

		if executionError != nil {
			response["error"] = executionError.Error()
			c.JSON(http.StatusInternalServerError, response)
		} else {
			response["result"] = executionResult
			c.JSON(http.StatusOK, response)
		}
	}
}

// Action execution functions

func executeTaskRedistribution(db *gorm.DB, action models.AgentAction) (map[string]interface{}, error) {
	// Get overloaded assignee
	fromUserID, _ := uuid.Parse(action.Data["fromUserId"].(string))
	toUserID, _ := uuid.Parse(action.Data["toUserId"].(string))
	taskCount := int(action.Data["taskCount"].(float64))

	// Find tasks to redistribute
	var tasks []models.Task
	err := db.Where("assignee_id = ? AND completed_at IS NULL", fromUserID).
		Limit(taskCount).
		Find(&tasks).Error
	if err != nil {
		return nil, err
	}

	redistributedTasks := make([]uuid.UUID, 0)
	for _, task := range tasks {
		task.AssigneeID = &toUserID
		if err := db.Save(&task).Error; err != nil {
			continue // Skip failed updates
		}
		redistributedTasks = append(redistributedTasks, task.ID)
	}

	return map[string]interface{}{
		"redistributedTasks": redistributedTasks,
		"fromUserId":         fromUserID,
		"toUserId":           toUserID,
		"count":              len(redistributedTasks),
	}, nil
}

func executeWIPAdjustment(db *gorm.DB, action models.AgentAction) (map[string]interface{}, error) {
	columnID, _ := uuid.Parse(action.Data["columnId"].(string))
	newLimit := int(action.Data["newLimit"].(float64))

	var column models.Column
	if err := db.First(&column, columnID).Error; err != nil {
		return nil, err
	}

	oldLimit := column.WIPLimit
	column.WIPLimit = newLimit
	if err := db.Save(&column).Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"columnId":  columnID,
		"oldLimit":  oldLimit,
		"newLimit":  newLimit,
	}, nil
}

func executeSubtaskCreation(db *gorm.DB, action models.AgentAction) (map[string]interface{}, error) {
	parentTaskID, _ := uuid.Parse(action.Data["parentTaskId"].(string))
	subtasks := action.Data["subtasks"].([]interface{})

	var parentTask models.Task
	if err := db.First(&parentTask, parentTaskID).Error; err != nil {
		return nil, err
	}

	createdSubtasks := make([]uuid.UUID, 0)
	for _, subtaskData := range subtasks {
		subtask := subtaskData.(map[string]interface{})
		
		newTask := models.Task{
			BoardID:     parentTask.BoardID,
			ColumnID:    parentTask.ColumnID,
			Title:       subtask["title"].(string),
			Description: subtask["description"].(string),
			AssigneeID:  parentTask.AssigneeID,
			Priority:    "medium",
			StoryPoints: 1,
		}

		if err := db.Create(&newTask).Error; err != nil {
			continue // Skip failed creations
		}
		createdSubtasks = append(createdSubtasks, newTask.ID)
	}

	return map[string]interface{}{
		"parentTaskId":     parentTaskID,
		"createdSubtasks":  createdSubtasks,
		"count":            len(createdSubtasks),
	}, nil
}

func executeOverdueReassignment(db *gorm.DB, action models.AgentAction) (map[string]interface{}, error) {
	// Find overdue tasks
	var overdueTasks []models.Task
	err := db.Where("due_date < ? AND completed_at IS NULL", time.Now()).
		Find(&overdueTasks).Error
	if err != nil {
		return nil, err
	}

	// Get available users (simplified logic)
	var users []models.User
	db.Find(&users)

	reassignedTasks := make([]uuid.UUID, 0)
	for _, task := range overdueTasks {
		// Simple round-robin assignment
		if len(users) > 0 {
			newAssignee := users[len(reassignedTasks)%len(users)]
			task.AssigneeID = &newAssignee.ID
			if err := db.Save(&task).Error; err != nil {
				continue
			}
			reassignedTasks = append(reassignedTasks, task.ID)
		}
	}

	return map[string]interface{}{
		"reassignedTasks": reassignedTasks,
		"count":           len(reassignedTasks),
	}, nil
}
