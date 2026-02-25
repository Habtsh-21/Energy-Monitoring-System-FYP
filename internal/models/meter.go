package models

import (
	"energy-monitoring-system/internal/db"
	"time"
)




type Meter struct {
	ID       uint   `gorm:"primaryKey"`
	SerialNo string `gorm:"uniqueIndex"`

	UserID *uint   
	Status string  `gorm:"default:unassigned"`

	CreatedAt time.Time
	UpdatedAt time.Time
}




func(meter *Meter) Create() error {
	if err := db.DB.Create(meter).Error; err != nil {
		return err
	}
	return nil
}

func(meter *Meter) Update() error {
	if err := db.DB.Save(meter).Error; err != nil {
		return err
	}
	return nil
}

func(meter *Meter) Delete() error {
	if err := db.DB.Delete(meter).Error; err != nil {
		return err
	}
	return nil
}

func GetMeter(meterId uint) (*Meter, error) {
	var meter Meter
	if err := db.DB.Where("ID = ?", meterId).First(&meter).Error; err != nil {
		return nil, err
	}
	return &meter, nil
}

func GetAllMeter() ([]Meter, error) {
	var meters []Meter
	if err := db.DB.Find(&meters).Error; err != nil {
		return nil, err
	}
	return meters, nil
}