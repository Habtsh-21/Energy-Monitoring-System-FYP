package models

import (
	"energy-monitoring-system/internal/db"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Record struct {
	BaseModel
	ID                uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID            uuid.UUID  `gorm:"type:uuid;index" json:"user_id"`
	MeterID           uuid.UUID  `gorm:"type:uuid;index" json:"meter_id"`
	AssignedAt        time.Time  `gorm:"column:assigned_at" json:"assigned_at"`
	UnassignedAt      *time.Time `gorm:"column:unassigned_at" json:"unassigned_at"`
	AssignedBy        string     `gorm:"column:assigned_by" json:"assigned_by"`
	IsCurrent         bool       `gorm:"column:is_current" json:"is_current"`
	TerminationReason string     `gorm:"column:termination_reason" json:"termination_reason"`

	User  *User  `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
	Meter *Meter `gorm:"foreignKey:MeterID;references:ID" json:"meter,omitempty"`
}

func (record *Record) Create(tx *gorm.DB) error {
	if tx == nil {
		tx = db.DB
	}
	if err := tx.Create(record).Error; err != nil {
		return err
	}
	return nil
}

func (record *Record) Update() error {
	if err := db.DB.Save(record).Error; err != nil {
		return err
	}
	return nil
}

func (record *Record) Delete() error {
	if err := db.DB.Delete(record).Error; err != nil {
		return err
	}
	return nil
}

func GetRecord(recordId uuid.UUID) (*Record, error) {
	var record Record
	if err := db.DB.Where("ID = ?", recordId).First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func GetAllRecord() ([]Record, error) {
	var records []Record
	if err := db.DB.Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}
