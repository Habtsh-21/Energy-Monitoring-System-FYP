package models

import (
	"energy-monitoring-system/internal/db"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MeterReading struct {
	BaseModel
	MeterID    uuid.UUID `gorm:"type:uuid;not null;index" json:"meter_id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	ReadingKWh float64   `gorm:"not null" json:"reading_kwh"`
	ReadAt     time.Time `gorm:"not null;index" json:"read_at"`
	Note       string    `gorm:"size:255" json:"note"`

	Meter *Meter `gorm:"foreignKey:MeterID" json:"meter,omitempty"`
	User  *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (MeterReading) TableName() string {
	return "meter_readings"
}

func (reading *MeterReading) Create(tx *gorm.DB) error {
	if tx == nil {
		tx = db.DB
	}
	if err := tx.Create(reading).Error; err != nil {
		return err
	}
	return nil
}

func (reading *MeterReading) Update() error {
	if err := db.DB.Save(reading).Error; err != nil {
		return err
	}
	return nil
}

func (reading *MeterReading) Delete() error {
	if err := db.DB.Delete(reading).Error; err != nil {
		return err
	}
	return nil
}

func GetMeterReading(readingID uuid.UUID) (*MeterReading, error) {
	var reading MeterReading
	if err := db.DB.Where("id = ?", readingID).First(&reading).Error; err != nil {
		return nil, err
	}
	return &reading, nil
}

func GetAllMeterReading() ([]MeterReading, error) {
	var readings []MeterReading
	if err := db.DB.Find(&readings).Error; err != nil {
		return nil, err
	}
	return readings, nil
}

func GetAllMeterReadingByMeterID(meterID uuid.UUID) ([]MeterReading, error) {
	var readings []MeterReading
	if err := db.DB.Where("meter_id = ?", meterID).Order("read_at desc").Find(&readings).Error; err != nil {
		return nil, err
	}
	return readings, nil
}

func GetMeterReadingByUserID(userID uuid.UUID) ([]MeterReading, error) {
	var readings []MeterReading
	if err := db.DB.Where("user_id = ?", userID).Order("read_at desc").Find(&readings).Error; err != nil {
		return nil, err
	}
	return readings, nil
}

func GetReadingUserPerMeter(meterID uuid.UUID, userID uuid.UUID) ([]MeterReading, error) {
	var readings []MeterReading
	if err := db.DB.Where("meter_id = ? AND user_id = ?", meterID, userID).Order("read_at desc").Find(&readings).Error; err != nil {
		return nil, err
	}
	return readings, nil
}