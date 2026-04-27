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


func GetAllMeterReadings(startTime, endTime time.Time, limit, offset int) ([]MeterReading, int64, error) {
	var readings []MeterReading
	var total int64
	baseQuery := db.DB.Model(&MeterReading{}).Where("read_at BETWEEN ? AND ?", startTime, endTime)

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := baseQuery.
		Order("read_at desc").
		Limit(limit).
		Offset(offset).
		Find(&readings).Error; err != nil {
		return nil, 0, err
	}

	return readings, total, nil
}


func GetMeterReadingByMeterID(meterID uuid.UUID,startTime,endTime time.Time,limit, offset int) ([]MeterReading, int64, error) {

	var readings []MeterReading
	var total int64
	baseQuery := db.DB.
		Model(&MeterReading{}).
		Where("meter_id = ? AND read_at BETWEEN ? AND ?", meterID, startTime, endTime)
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := baseQuery.
		Order("read_at desc").
		Limit(limit).
		Offset(offset).
		Find(&readings).Error; err != nil {
		return nil, 0, err
	}

	return readings, total, nil
}
func GetMeterReadingByUserID(userID uuid.UUID,startTime,endTime time.Time,limit, offset int) ([]MeterReading, int64, error) {

	var readings []MeterReading
	var total int64
	baseQuery := db.DB.
		Model(&MeterReading{}).
		Where("user_id = ? AND read_at BETWEEN ? AND ?", userID, startTime, endTime)
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := baseQuery.
		Order("read_at desc").
		Limit(limit).
		Offset(offset).
		Find(&readings).Error; err != nil {
		return nil, 0, err
	}

	return readings, total, nil
}

func GetRecentMeterReadingsByMeterID(meterID uuid.UUID, limit int) ([]MeterReading, error) {
	var readings []MeterReading
	if err := db.DB.Where("meter_id = ?", meterID).
		Order("read_at desc").
		Limit(limit).
		Find(&readings).Error; err != nil {
		return nil, err
	}
	return readings, nil
}
