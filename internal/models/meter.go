package models

import (
	"energy-monitoring-system/internal/db"
	
	"strings"

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
	IsAvailable       bool   `gorm:"default:true;index" json:"is_available"`
	RelayStatus       string `gorm:"size:20;default:'OFF'" json:"relay_status"` // ON, OFF

	Record      []Record      `gorm:"foreignKey:MeterID;" json:"record,omitempty"`
	LineReading []LineReading `gorm:"foreignKey:MeterID;" json:"line_reading,omitempty"`
}

type MeterRegisterRequest struct {
	MeterSerialNumber string `json:"meter_serial_number"`
	MeterType         string `json:"meter_type"`
	Manufacturer      string `json:"manufacturer"`
	Model             string `json:"model"`
	FirmwareVersion   string `json:"firmware_version"`
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

func UpdateMeterParameters(tx *gorm.DB, meterID uuid.UUID, updates map[string]any) error {
	if tx == nil {
		tx = db.DB
	}
	if err := tx.Model(&Meter{}).Where("id = ?", meterID).Updates(updates).Error; err != nil {
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

func GetMeterBySerialNo(serialNo string) (*Meter, error) {
	


	var meter Meter
	if err := db.DB.Where("meter_serial_number = ?", serialNo).First(&meter).Error; err != nil {
		return nil, err
	}
	return &meter, nil
}

func GetMeterByID(id uuid.UUID) (*Meter, error) {
	var meter Meter
	if err := db.DB.Preload("Record").
		Preload("LineReading").
		Where("ID = ?", id).First(&meter).Error; err != nil {
		return nil, err
	}
	return &meter, nil
}

func GetMeterIDBySerialNo(serialNo string) (uuid.UUID, error) {
	
	var meter Meter
	if err := db.DB.Where("meter_serial_number = ?", serialNo).First(&meter).Error; err != nil {
		return uuid.Nil, err
	}
	return meter.ID, nil
}

func GetSerialNumber(meterID uuid.UUID) (string, error) {
	var meter Meter
	if err := db.DB.Where("id = ?", meterID).First(&meter).Error; err != nil {
		return "", err
	}
	return meter.MeterSerialNumber, nil
}

func GetAllMeter() ([]Meter, error) {
	var meters []Meter
	if err := db.DB.Find(&meters).Error; err != nil {
		return nil, err
	}
	return meters, nil
}

func CheckSerialNo(serialNo string) bool {
	serialNo = normalizeMeterSerial(serialNo)
	if serialNo == "" {
		return false
	}
	var meter Meter
	if err := db.DB.Where("meter_serial_number = ?", serialNo).First(&meter).Error; err != nil {
		return false
	}
	return true
}

func normalizeMeterSerial(serialNo string) string {
	return strings.TrimSpace(serialNo)
}

func IsMeterAvailable(meterID uuid.UUID) (bool, error) {
	var meter Meter
	if err := db.DB.Where("id = ?", meterID).First(&meter).Error; err != nil {
		return false, err
	}
	if meter.IsAvailable == false {
		return false, nil
	}
	return true, nil
}

func IsMeterAssigned(meterID uuid.UUID) (bool, error) {
	var meter Meter
	if err := db.DB.Where("id = ?", meterID).First(&meter).Error; err != nil {
		return false, err
	}
	if meter.IsAvailable == true {
		return false, nil
	}
	return true, nil
}
