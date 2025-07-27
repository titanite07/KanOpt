package api

import (
	"net/http"
	"time"
	"kanopt/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VelocityResponse struct {
	CurrentVelocity   float64                `json:"currentVelocity"`
	AverageVelocity   float64                `json:"averageVelocity"`
	VelocityTrend     string                 `json:"velocityTrend"`
	WeeklyMetrics     []models.VelocityMetric `json:"weeklyMetrics"`
	PredictedVelocity float64                `json:"predictedVelocity"`
}

type BurndownData struct {
	Date           string  `json:"date"`
	Remaining      int     `json:"remaining"`
	Ideal          int     `json:"ideal"`
	Actual         int     `json:"actual"`
	TotalStoryPoints int   `json:"totalStoryPoints"`
}

type RiskTrendData struct {
	Date      string  `json:"date"`
	HighRisk  int     `json:"highRisk"`
	MediumRisk int    `json:"mediumRisk"`
	LowRisk   int     `json:"lowRisk"`
	TotalTasks int    `json:"totalTasks"`
}

type TeamPerformanceData struct {
	UserID          uuid.UUID `json:"userId"`
	Name            string    `json:"name"`
	Avatar          string    `json:"avatar"`
	CompletedTasks  int       `json:"completedTasks"`
	TotalStoryPoints int      `json:"totalStoryPoints"`
	AverageCycleTime float64  `json:"averageCycleTime"`
	Velocity        float64   `json:"velocity"`
	EfficiencyScore float64   `json:"efficiencyScore"`
}

func GetVelocityMetrics(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		boardID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID"})
			return
		}

		// Get velocity metrics for the last 12 weeks
		var metrics []models.VelocityMetric
		err = db.Where("board_id = ?", boardID).
			Order("sprint_week DESC").
			Limit(12).
			Find(&metrics).Error
		
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Calculate current and average velocity
		var currentVelocity, averageVelocity, totalVelocity float64
		if len(metrics) > 0 {
			currentVelocity = metrics[0].Velocity
			for _, metric := range metrics {
				totalVelocity += metric.Velocity
			}
			averageVelocity = totalVelocity / float64(len(metrics))
		}

		// Determine trend
		trend := "stable"
		if len(metrics) >= 2 {
			if currentVelocity > metrics[1].Velocity*1.1 {
				trend = "increasing"
			} else if currentVelocity < metrics[1].Velocity*0.9 {
				trend = "decreasing"
			}
		}

		// Predict next velocity (simple linear trend)
		predictedVelocity := currentVelocity
		if len(metrics) >= 3 {
			// Calculate slope of last 3 weeks
			slope := (metrics[0].Velocity - metrics[2].Velocity) / 2
			predictedVelocity = currentVelocity + slope
		}

		response := VelocityResponse{
			CurrentVelocity:   currentVelocity,
			AverageVelocity:   averageVelocity,
			VelocityTrend:     trend,
			WeeklyMetrics:     metrics,
			PredictedVelocity: predictedVelocity,
		}

		c.JSON(http.StatusOK, response)
	}
}

func GetBurndownData(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		boardID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID"})
			return
		}

		// Get sprint duration (default to 2 weeks)
		sprintDays := 14
		sprintStart := time.Now().AddDate(0, 0, -sprintDays)

		// Get total story points at sprint start
		var totalStoryPoints int
		db.Model(&models.Task{}).
			Where("board_id = ? AND created_at <= ?", boardID, sprintStart).
			Select("COALESCE(SUM(story_points), 0)").
			Scan(&totalStoryPoints)

		var burndownData []BurndownData
		
		// Generate burndown data for each day
		for i := 0; i <= sprintDays; i++ {
			currentDate := sprintStart.AddDate(0, 0, i)
			
			// Calculate remaining points
			var completedPoints int
			db.Model(&models.Task{}).
				Where("board_id = ? AND completed_at <= ?", boardID, currentDate).
				Select("COALESCE(SUM(story_points), 0)").
				Scan(&completedPoints)
			
			remaining := totalStoryPoints - completedPoints
			ideal := totalStoryPoints - (totalStoryPoints * i / sprintDays)
			
			burndownData = append(burndownData, BurndownData{
				Date:             currentDate.Format("2006-01-02"),
				Remaining:        remaining,
				Ideal:            ideal,
				Actual:           remaining,
				TotalStoryPoints: totalStoryPoints,
			})
		}

		c.JSON(http.StatusOK, burndownData)
	}
}

func GetRiskTrends(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		boardID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID"})
			return
		}

		// Get risk trends for the last 30 days
		var riskTrends []RiskTrendData
		
		for i := 29; i >= 0; i-- {
			currentDate := time.Now().AddDate(0, 0, -i)
			dateStr := currentDate.Format("2006-01-02")
			
			// Count risks by level for this date
			var highRisk, mediumRisk, lowRisk int64
			
			db.Model(&models.RiskPrediction{}).
				Where("board_id = ? AND DATE(created_at) = ? AND level = ?", boardID, dateStr, "high").
				Count(&highRisk)
			
			db.Model(&models.RiskPrediction{}).
				Where("board_id = ? AND DATE(created_at) = ? AND level = ?", boardID, dateStr, "medium").
				Count(&mediumRisk)
			
			db.Model(&models.RiskPrediction{}).
				Where("board_id = ? AND DATE(created_at) = ? AND level = ?", boardID, dateStr, "low").
				Count(&lowRisk)
			
			// Count total tasks
			var totalTasks int64
			db.Model(&models.Task{}).
				Where("board_id = ? AND DATE(created_at) <= ?", boardID, dateStr).
				Count(&totalTasks)
			
			riskTrends = append(riskTrends, RiskTrendData{
				Date:       dateStr,
				HighRisk:   int(highRisk),
				MediumRisk: int(mediumRisk),
				LowRisk:    int(lowRisk),
				TotalTasks: int(totalTasks),
			})
		}

		c.JSON(http.StatusOK, riskTrends)
	}
}

func GetTeamPerformance(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		boardID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID"})
			return
		}

		// Get all users who have tasks in this board
		var users []models.User
		err = db.Joins("JOIN tasks ON users.id = tasks.assignee_id").
			Where("tasks.board_id = ?", boardID).
			Group("users.id").
			Find(&users).Error
		
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var teamPerformance []TeamPerformanceData
		
		for _, user := range users {
			// Count completed tasks
			var completedTasks int64
			db.Model(&models.Task{}).
				Where("board_id = ? AND assignee_id = ? AND completed_at IS NOT NULL", boardID, user.ID).
				Count(&completedTasks)
			
			// Sum story points
			var totalStoryPoints int
			db.Model(&models.Task{}).
				Where("board_id = ? AND assignee_id = ? AND completed_at IS NOT NULL", boardID, user.ID).
				Select("COALESCE(SUM(story_points), 0)").
				Scan(&totalStoryPoints)
			
			// Calculate average cycle time
			var tasks []models.Task
			db.Where("board_id = ? AND assignee_id = ? AND completed_at IS NOT NULL", boardID, user.ID).
				Find(&tasks)
			
			var totalCycleTime float64
			for _, task := range tasks {
				if task.CompletedAt != nil {
					cycleTime := task.CompletedAt.Sub(task.CreatedAt).Hours() / 24 // Days
					totalCycleTime += cycleTime
				}
			}
			
			averageCycleTime := float64(0)
			if len(tasks) > 0 {
				averageCycleTime = totalCycleTime / float64(len(tasks))
			}
			
			// Calculate velocity (story points per week)
			velocity := float64(0)
			if len(tasks) > 0 {
				weeksSinceFirstTask := time.Since(tasks[len(tasks)-1].CreatedAt).Hours() / (24 * 7)
				if weeksSinceFirstTask > 0 {
					velocity = float64(totalStoryPoints) / weeksSinceFirstTask
				}
			}
			
			// Calculate efficiency score (arbitrary formula)
			efficiencyScore := float64(0)
			if averageCycleTime > 0 {
				efficiencyScore = velocity / averageCycleTime * 10
				if efficiencyScore > 100 {
					efficiencyScore = 100
				}
			}
			
			teamPerformance = append(teamPerformance, TeamPerformanceData{
				UserID:           user.ID,
				Name:             user.Name,
				Avatar:           user.Avatar,
				CompletedTasks:   int(completedTasks),
				TotalStoryPoints: totalStoryPoints,
				AverageCycleTime: averageCycleTime,
				Velocity:         velocity,
				EfficiencyScore:  efficiencyScore,
			})
		}

		c.JSON(http.StatusOK, teamPerformance)
	}
}
