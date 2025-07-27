package database

import (
	"kanopt/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Board{},
		&models.Column{},
		&models.Task{},
		&models.Event{},
		&models.User{},
		&models.AgentAction{},
		&models.Suggestion{},
		&models.RiskPrediction{},
		&models.VelocityMetric{},
	)
}
