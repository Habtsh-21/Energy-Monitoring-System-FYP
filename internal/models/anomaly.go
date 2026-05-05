package models

import (
	"energy-monitoring-system/internal/db"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	AnomalyStatusOpen     = "open"
	AnomalyStatusResolved = "resolved"

	AnomalyTypeRollback  = "rollback_reading"
	AnomalyTypeFlatUsage = "flat_usage"
)

type Anomaly struct {
	BaseModel
	ReadingID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"reading_id"`
	Type           string     `gorm:"size:50;not null;index" json:"type"`
	Reason         string     `gorm:"size:500;not null" json:"reason"`
	Status         string     `gorm:"size:20;not null;default:open;index" json:"status"`
	DetectedAt     time.Time  `gorm:"not null;index" json:"detected_at"`
	ResolvedAt     *time.Time `json:"resolved_at"`
	ResolvedBy     string     `gorm:"size:100" json:"resolved_by"`
	ResolutionNote string     `gorm:"size:500" json:"resolution_note"`

	LineReading *LineReading `gorm:"foreignKey:ReadingID" json:"line_reading,omitempty"`
}

func (anomaly *Anomaly) Create() error {

	return db.DB.Create(anomaly).Error
}

func GetAnomalies(status string) ([]Anomaly, error) {
	var anomalies []Anomaly
	query := db.DB.Preload("LineReading").Order("detected_at desc")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Find(&anomalies).Error; err != nil {
		return nil, err
	}
	return anomalies, nil
}

func ResolveAnomaly(anomalyID uuid.UUID, resolvedBy, note string) error {
	resolvedAt := time.Now()
	updates := map[string]any{
		"status":          AnomalyStatusResolved,
		"resolved_at":     &resolvedAt,
		"resolved_by":     resolvedBy,
		"resolution_note": note,
		"updated_at":      resolvedAt,
	}

	result := db.DB.Model(&Anomaly{}).
		Where("id = ? AND status = ?", anomalyID, AnomalyStatusOpen).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
