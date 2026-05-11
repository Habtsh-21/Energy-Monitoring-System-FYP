package models

import (
	"energy-monitoring-system/internal/db"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)


type Anomaly struct {
	BaseModel
	ReadingID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"reading_id"`
	Reason         string     `gorm:"size:500;not null" json:"reason"`
	DetectedAt     time.Time  `gorm:"not null;index" json:"detected_at"`
	LineReading    *LineReading `gorm:"foreignKey:ReadingID" json:"line_reading,omitempty"`
}


func(anomaly *Anomaly) CreateAnomaly(tx *gorm.DB) error {
	if tx == nil {
		tx = db.DB
	}
		return tx.Create(&anomaly).Error
}

func GetAnomalies() ([]Anomaly, error) {
	var anomalies []Anomaly
	query := db.DB.Preload("LineReading")
	if err := query.Find(&anomalies).Error; err != nil {
		return nil, err
	}
	return anomalies, nil
}


