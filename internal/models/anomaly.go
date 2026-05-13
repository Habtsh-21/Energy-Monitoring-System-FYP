package models

import (
	"energy-monitoring-system/internal/db"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Anomaly struct {
	BaseModel
	ReadingID      uuid.UUID    `gorm:"type:uuid;not null;index" json:"reading_id"`
	Type           string       `gorm:"column:type;type:varchar(20);not null;index" json:"type"`
	Verdict        string       `gorm:"type:varchar(20);not null;index" json:"verdict"`
	MeterClaim     string       `gorm:"type:varchar(20)" json:"meter_claim"`   // what the meter reported
	OurDetection   string       `gorm:"type:varchar(20)" json:"our_detection"` // what our analysis found
	Conflict       bool         `json:"conflict"`                              // true when the two disagree
	ConflictReason string       `json:"conflict_reason,omitempty"`
	Signals        []string     `gorm:"type:jsonb;serializer:json" json:"signals,omitempty"`
	Reason         string       `json:"reason,omitempty"`
	DetectedAt     time.Time    `gorm:"not null;index" json:"detected_at"`
	LineReading    *LineReading `gorm:"foreignKey:ReadingID" json:"line_reading,omitempty"`
}

type AnomalyResponse struct {
	ID         uuid.UUID `json:"id"`
	Type       string    `json:"type"`
	Verdict    string    `json:"verdict"`
	Signals    []string  `gorm:"type:jsonb;serializer:json" json:"signals"`
	DetectedAt time.Time `json:"detected_at"`
}

func (anomaly *Anomaly) Create(tx *gorm.DB) error {
	if tx == nil {
		tx = db.DB
	}
	return tx.Create(anomaly).Error
}

func GetAnomalies() ([]AnomalyResponse, error) {
	anomalies := make([]AnomalyResponse, 0)
	err := db.DB.
		Model(&Anomaly{}).
		Select("id", "type", "verdict", "signals", "detected_at").
		Scan(&anomalies).Error
	if err != nil {
		return nil, err
	}
	return anomalies, nil
}

func GetAnomalyByID(id uuid.UUID) (Anomaly, error) {
	var anomaly Anomaly
	err := db.DB.
		Preload("LineReading").
		Where("id = ?", id).
		First(&anomaly).Error
	if err != nil {
		return anomaly, err
	}
	return anomaly, nil
}
