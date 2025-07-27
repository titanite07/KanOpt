package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"kanopt/internal/models"
	"kanopt/internal/messaging"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PredictionRequest struct {
	TimeHorizon string `json:"timeHorizon"` // "1week", "2weeks", "1month"
	Metrics     []string `json:"metrics"`   // "velocity", "completion", "risk"
}

type PredictionResponse struct {
	BoardID        uuid.UUID                 `json:"boardId"`
	TimeHorizon    string                    `json:"timeHorizon"`
	Predictions    map[string]interface{}    `json:"predictions"`
	Confidence     float64                   `json:"confidence"`
	GeneratedAt    string                    `json:"generatedAt"`
	ModelVersion   string                    `json:"modelVersion"`
}

type RiskAnalysisRequest struct {
	TaskIDs []uuid.UUID `json:"taskIds"`
	Factors []string    `json:"factors"` // "deadline", "complexity", "assignee_workload"
}

type RiskAnalysisResponse struct {
	BoardID     uuid.UUID                    `json:"boardId"`
	Risks       []models.RiskPrediction      `json:"risks"`
	Summary     map[string]interface{}       `json:"summary"`
	Recommendations []string                 `json:"recommendations"`
}

func GetPredictions(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		boardID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID"})
			return
		}

		var request PredictionRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			// Use defaults if no body provided
			request = PredictionRequest{
				TimeHorizon: "2weeks",
				Metrics:     []string{"velocity", "completion", "risk"},
			}
		}

		// Get historical data for predictions
		var velocityMetrics []models.VelocityMetric
		db.Where("board_id = ?", boardID).
			Order("sprint_week DESC").
			Limit(12).
			Find(&velocityMetrics)

		var tasks []models.Task
		db.Where("board_id = ?", boardID).Find(&tasks)

		// Prepare data for AI service
		aiRequest := map[string]interface{}{
			"boardId":         boardID,
			"timeHorizon":     request.TimeHorizon,
			"metrics":         request.Metrics,
			"velocityHistory": velocityMetrics,
			"currentTasks":    tasks,
		}

		// Call AI service
		predictions, err := callAIService("/api/predict", aiRequest)
		if err != nil {
			// Fallback to simple predictions if AI service is unavailable
			predictions = generateFallbackPredictions(velocityMetrics, tasks, request.TimeHorizon)
		}

		response := PredictionResponse{
			BoardID:      boardID,
			TimeHorizon:  request.TimeHorizon,
			Predictions:  predictions,
			Confidence:   0.75,
			GeneratedAt:  "2024-01-15T10:30:00Z",
			ModelVersion: "v1.2.0",
		}

		c.JSON(http.StatusOK, response)
	}
}

func AnalyzeRisk(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		boardID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID"})
			return
		}

		var request RiskAnalysisRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			// Analyze all tasks if none specified
			var allTasks []models.Task
			db.Where("board_id = ? AND completed_at IS NULL", boardID).Find(&allTasks)
			
			for _, task := range allTasks {
				request.TaskIDs = append(request.TaskIDs, task.ID)
			}
			request.Factors = []string{"deadline", "complexity", "assignee_workload"}
		}

		var tasks []models.Task
		db.Where("id IN ?", request.TaskIDs).Preload("Assignee").Find(&tasks)

		// Prepare data for AI service
		aiRequest := map[string]interface{}{
			"boardId": boardID,
			"tasks":   tasks,
			"factors": request.Factors,
		}

		// Call AI service for risk analysis
		aiResponse, err := callAIService("/api/analyze-risk", aiRequest)
		if err != nil {
			// Fallback to rule-based risk analysis
			aiResponse = generateFallbackRiskAnalysis(tasks, request.Factors)
		}

		// Create risk predictions in database
		var risks []models.RiskPrediction
		if riskData, ok := aiResponse["risks"].([]interface{}); ok {
			for _, risk := range riskData {
				if riskMap, ok := risk.(map[string]interface{}); ok {
					riskPrediction := models.RiskPrediction{
						BoardID:     boardID,
						Type:        riskMap["type"].(string),
						Level:       riskMap["level"].(string),
						Score:       riskMap["score"].(float64),
						Description: riskMap["description"].(string),
						Data:        riskMap,
					}
					
					if taskID, exists := riskMap["taskId"]; exists {
						if taskUUID, err := uuid.Parse(taskID.(string)); err == nil {
							riskPrediction.TaskID = &taskUUID
						}
					}
					
					db.Create(&riskPrediction)
					risks = append(risks, riskPrediction)
				}
			}
		}

		response := RiskAnalysisResponse{
			BoardID:     boardID,
			Risks:       risks,
			Summary:     aiResponse["summary"].(map[string]interface{}),
			Recommendations: []string{
				"Consider redistributing tasks from overloaded team members",
				"Review tasks approaching deadlines for scope reduction",
				"Add more specific requirements to complex tasks",
			},
		}

		c.JSON(http.StatusOK, response)
	}
}

// Helper functions

func callAIService(endpoint string, data map[string]interface{}) (map[string]interface{}, error) {
	// TODO: Get AI service URL from config
	aiServiceURL := "http://localhost:8000"
	
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(aiServiceURL+endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI service returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func generateFallbackPredictions(metrics []models.VelocityMetric, tasks []models.Task, timeHorizon string) map[string]interface{} {
	predictions := make(map[string]interface{})
	
	// Calculate simple velocity prediction
	if len(metrics) > 0 {
		currentVelocity := metrics[0].Velocity
		predictions["velocity"] = map[string]interface{}{
			"predicted": currentVelocity * 1.05, // Slight improvement
			"range": map[string]float64{
				"min": currentVelocity * 0.8,
				"max": currentVelocity * 1.2,
			},
		}
	}

	// Calculate completion prediction
	activeTasks := 0
	totalStoryPoints := 0
	for _, task := range tasks {
		if task.CompletedAt == nil {
			activeTasks++
			totalStoryPoints += task.StoryPoints
		}
	}

	daysInHorizon := 14 // Default 2 weeks
	switch timeHorizon {
	case "1week":
		daysInHorizon = 7
	case "1month":
		daysInHorizon = 30
	}

	expectedCompletion := float64(activeTasks) * 0.7 // Assume 70% completion rate
	predictions["completion"] = map[string]interface{}{
		"expectedTasks":  int(expectedCompletion),
		"totalTasks":     activeTasks,
		"storyPoints":    int(float64(totalStoryPoints) * 0.7),
		"completionRate": 0.7,
	}

	// Risk prediction
	predictions["risk"] = map[string]interface{}{
		"overallRisk": "medium",
		"riskFactors": []string{
			"High work in progress",
			"Approaching deadlines",
		},
		"riskScore": 0.6,
	}

	return predictions
}

func generateFallbackRiskAnalysis(tasks []models.Task, factors []string) map[string]interface{} {
	risks := make([]interface{}, 0)
	
	for _, task := range tasks {
		riskScore := 0.0
		riskLevel := "low"
		riskFactors := make([]string, 0)
		
		// Check deadline factor
		if task.DueDate != nil && task.DueDate.Before(task.CreatedAt.AddDate(0, 0, 7)) {
			riskScore += 0.3
			riskFactors = append(riskFactors, "tight_deadline")
		}
		
		// Check complexity (based on story points)
		if task.StoryPoints > 8 {
			riskScore += 0.2
			riskFactors = append(riskFactors, "high_complexity")
		}
		
		// Check if unassigned
		if task.AssigneeID == nil {
			riskScore += 0.3
			riskFactors = append(riskFactors, "unassigned")
		}
		
		// Determine risk level
		if riskScore >= 0.7 {
			riskLevel = "high"
		} else if riskScore >= 0.4 {
			riskLevel = "medium"
		}
		
		if riskScore > 0.1 {
			risks = append(risks, map[string]interface{}{
				"taskId":      task.ID.String(),
				"type":        "delivery_risk",
				"level":       riskLevel,
				"score":       riskScore,
				"description": fmt.Sprintf("Task '%s' has elevated risk factors", task.Title),
				"factors":     riskFactors,
			})
		}
	}
	
	summary := map[string]interface{}{
		"totalTasks":    len(tasks),
		"riskyTasks":    len(risks),
		"averageRisk":   0.3,
		"highRiskCount": 0,
	}
	
	return map[string]interface{}{
		"risks":   risks,
		"summary": summary,
	}
}
