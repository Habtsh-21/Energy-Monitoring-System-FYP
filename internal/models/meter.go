package models

import (
	"energy-monitoring-system/internal/db"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Meter struct {
	BaseModel
	MeterSerialNumber string `gorm:"column:meter_serial_number;uniqueIndex;size:50;not null" json:"meter_serial_number"`
	MeterType         string `gorm:"column:meter_type;size:50;not null" json:"meter_type"` // single_phase, three_phase
	Manufacturer      string `gorm:"size:100" json:"manufacturer"`
	Model             string `gorm:"size:100" json:"model"`
	FirmwareVersion   string `gorm:"column:firmware_version;size:50" json:"firmware_version"`
	Status            string `gorm:"size:20;default:'available';index" json:"status"` // available, assigned, maintenance, retired

	Record []Record `gorm:"foreignKey:MeterID" json:"record,omitempty"`
}

func (meter *Meter) Create() error {
	if err := db.DB.Create(meter).Error; err != nil {
		return err
	}
	return nil
}

func (meter *Meter) Update() error {
	if err := db.DB.Save(meter).Error; err != nil {
		return err
	}
	return nil
}

func (meter *Meter) Delete() error {
	if err := db.DB.Delete(meter).Error; err != nil {
		return err
	}
	return nil
}

func GetMeter(serialNo string) (*Meter, error) {
	var meter Meter
	if err := db.DB.Where("meter_serial_number = ?", serialNo).First(&meter).Error; err != nil {
		return nil, err
	}
	return &meter, nil
}

func GetMeterID(serialNo string) (uuid.UUID, error) {
	var meter Meter
	if err := db.DB.Where("meter_serial_number = ?", serialNo).First(&meter).Error; err != nil {
		return uuid.Nil, err
	}
	return meter.ID, nil
}

func GetAllMeter() ([]Meter, error) {
	var meters []Meter
	if err := db.DB.Find(&meters).Error; err != nil {
		return nil, err
	}
	return meters, nil
}

func CheckMeterSerialNo(serialNo string) bool {
	var meter Meter
	if err := db.DB.Where("meter_serial_number = ?", serialNo).First(&meter).Error; err != nil {
		return false
	}
	return true
}

func CheckMeterId(id uuid.UUID) bool {
	var meter Meter
	if err := db.DB.Where("ID = ?", id).First(&meter).Error; err != nil {
		return false
	}
	return true
}

func UpdateMeterStatus(tx *gorm.DB, id uuid.UUID, status string) error {
	if tx == nil {
		tx = db.DB
	}

	if err := tx.Model(&Meter{}).
		Where("id = ?", id).
		Update("status", status).Error; err != nil {
		return err
	}

	return nil
}
