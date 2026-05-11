package models

import (
	"energy-monitoring-system/internal/db"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LineReading struct {
	BaseModel

	MeterID uuid.UUID `gorm:"type:uuid;not null;index" json:"meter_id"`
	UserID  uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`

	PoleCurrentA  float64 `gorm:"not null" json:"pole_current_a"`
	PoleVoltageV  float64 `gorm:"not null" json:"pole_voltage_v"`
	MeterCurrentA float64 `gorm:"not null" json:"meter_current_a"`
	MeterVoltageV float64 `gorm:"not null" json:"meter_voltage_v"`

	PoleApparentPowerVA  float64 `gorm:"not null" json:"pole_apparent_power_va"`
	MeterApparentPowerVA float64 `gorm:"not null" json:"meter_apparent_power_va"`
	DeltaCurrentA        float64 `gorm:"not null" json:"delta_current_a"`
	DeltaVoltageV        float64 `gorm:"not null" json:"delta_voltage_v"`
	PowerLossPct         float64 `gorm:"not null" json:"power_loss_pct"`

	RecordedAt time.Time `gorm:"not null;index" json:"recorded_at"`
	Note       string    `gorm:"size:255"       json:"note"`

	Meter *Meter `gorm:"foreignKey:MeterID" json:"meter,omitempty"`
	User  *User  `gorm:"foreignKey:UserID"  json:"user,omitempty"`
}



func (lr *LineReading) ComputeDerived() {
	lr.PoleApparentPowerVA = lr.PoleVoltageV * lr.PoleCurrentA
	lr.MeterApparentPowerVA = lr.MeterVoltageV * lr.MeterCurrentA
	lr.DeltaCurrentA = lr.PoleCurrentA - lr.MeterCurrentA
	lr.DeltaVoltageV = lr.PoleVoltageV - lr.MeterVoltageV

	if lr.PoleApparentPowerVA > 0 {
		lr.PowerLossPct = ((lr.PoleApparentPowerVA - lr.MeterApparentPowerVA) / lr.PoleApparentPowerVA) * 100
	}
}


type LineReadingRequest struct {
	MeterSerialNumber string  `json:"meter_serial_number"`
	LoopDuration      float64 `json:"loop_duration"`
	PoleCurrentA      float64 `json:"pole_current_a"`
	PoleVoltageV      float64 `json:"pole_voltage_v"`
	MeterCurrentA     float64 `json:"meter_current_a"`
	MeterVoltageV     float64 `json:"meter_voltage_v"`
	PolePowerVA       float64 `json:"pole_power_va"`
	MeterPowerVA      float64 `json:"meter_power_va"`
	ConsumedKwh       float64 `json:"consumed_kwh"`
	RemainingKwh      float64 `json:"remaining_kwh"`
	PowerLossPct      float64 `json:"power_loss_pct"`
	IsConnected       bool    `json:"is_connected"`
	BypassStatus      string  `json:"bypass_status"`
	SystemLocked      bool    `json:"system_locked"`
	RecordedAt        string  `json:"recorded_at"` 
	Note              string  `json:"note"`
}


type Severity string



var ( 
	SeverityNormal    Severity = "normal"
	SeveritySuspect   Severity = "suspect"
	SeverityConfirmed Severity = "confirmed"
)


type BypassResult struct {
	PowerLoss   float64
	CurrentLoss float64
	VoltageDrop float64
	Signals     []string
	Reason      string
	Severity    Severity
}



const (
	LineCurrentLowThreshold  float64 = 1.0
	LineCurrentHighThreshold float64 = 3.0
	LineVoltageLowThreshold  float64 = 5.0
	LineVoltageHighThreshold float64 = 15.0
	PowerLossLowThreshold    float64 = 5.0
	PowerLossHighThreshold   float64 = 15.0
)


func (lr *LineReading) Create(tx *gorm.DB) error {
	if tx == nil {
		tx = db.DB
	}
	return tx.Create(lr).Error
}
 

func GetLineReadingsByMeterID(meterID uuid.UUID, start, end time.Time, limit, offset int) ([]LineReading, int64, error) {
	var rows []LineReading
	var total int64

	base := db.DB.Model(&LineReading{}).Where("meter_id = ? AND recorded_at BETWEEN ? AND ?", meterID, start, end)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := base.Order("recorded_at asc").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}



func GetRecentLineReadings(meterID uuid.UUID, limit int) ([]LineReading, error) {
	var rows []LineReading
	err := db.DB.Where("meter_id = ?", meterID).Order("recorded_at desc").Limit(limit).Find(&rows).Error
	return rows, err
}

func GetAllActiveMeterIDs() ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := db.DB.Model(&LineReading{}).Distinct("meter_id").Pluck("meter_id", &ids).Error
	return ids, err
}
