package models

import (
	"energy-monitoring-system/internal/db"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrMeterDisabledByAdmin = errors.New("meter disabled by admin")
	ErrMeterDisabledByOwner = errors.New("meter disabled by owner")
)

type Meter struct {
	BaseModel
	MeterSerialNumber string `gorm:"column:meter_serial_number;uniqueIndex;size:50;not null" json:"meter_serial_number"`
	MeterType         string `gorm:"column:meter_type;size:50;not null" json:"meter_type"` // single_phase, three_phase
	Manufacturer      string `gorm:"size:100" json:"manufacturer"`
	Model             string `gorm:"size:100" json:"model"`
	FirmwareVersion   string `gorm:"column:firmware_version;size:50" json:"firmware_version"`
	IsAvailable       bool   `gorm:"default:true;index" json:"is_available"`
	RelayStatus       string `gorm:"size:20;default:'ON'" json:"relay_status"` // ON, OFF
	AdminDisabled     bool   `gorm:"default:false" json:"admin_disabled"`         // admin hard-override: always OFF when true
	OwnerDisabled     bool   `gorm:"default:false" json:"owner_disabled"`          // owner soft-override: owner can disable their own meter

	Record      []Record      `gorm:"foreignKey:MeterID;" json:"record,omitempty"`
	LineReading []LineReading `gorm:"foreignKey:MeterID;" json:"line_reading,omitempty"`
}

// MeterStatus holds only the fields the line handler needs to decide relay state.
type MeterStatus struct {
	ID            string
	AdminDisabled bool
	OwnerDisabled bool
	RelayStatus   string
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

// GetMeterStatus returns a lightweight status record — no relation preloads.
func GetMeterStatus(meterID uuid.UUID) (*MeterStatus, error) {
	var m Meter
	if err := db.DB.Select("id", "admin_disabled", "owner_disabled", "relay_status").
		Where("id = ?", meterID).First(&m).Error; err != nil {
		return nil, err
	}
	return &MeterStatus{
		ID:            m.ID.String(),
		AdminDisabled: m.AdminDisabled,
		OwnerDisabled: m.OwnerDisabled,
		RelayStatus:   m.RelayStatus,
	}, nil
}

func desiredRelayStatus(adminDisabled, ownerDisabled bool) string {
	if adminDisabled || ownerDisabled {
		return "OFF"
	}
	return "ON"
}

func SetMeterStatus(tx *gorm.DB, meterID uuid.UUID, isDisabled bool) error {
	if tx == nil {
		tx = db.DB
	}
	return tx.Transaction(func(innerTx *gorm.DB) error {
		var m Meter
		if err := innerTx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id", "admin_disabled", "owner_disabled").
			Where("id = ?", meterID).
			First(&m).Error; err != nil {
			return err
		}

		// Enabling by admin is blocked if the owner has disabled it.
		if !isDisabled && m.OwnerDisabled {
			return ErrMeterDisabledByOwner
		}

		relayStatus := desiredRelayStatus(isDisabled, m.OwnerDisabled)
		if err := innerTx.Model(&Meter{}).Where("id = ?", meterID).Updates(map[string]any{
			"admin_disabled": isDisabled,
			"relay_status":   relayStatus,
		}).Error; err != nil {
			return fmt.Errorf("update meter status: %w", err)
		}
		return nil
	})
}

// SetOwnerDisabled lets the meter owner enable or disable their own meter.
// When disabled the relay is commanded OFF; when re-enabled it returns to ON.
func SetOwnerDisabled(tx *gorm.DB, meterID uuid.UUID, disabled bool) error {
	if tx == nil {
		tx = db.DB
	}
	return tx.Transaction(func(innerTx *gorm.DB) error {
		var m Meter
		if err := innerTx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id", "admin_disabled", "owner_disabled").
			Where("id = ?", meterID).
			First(&m).Error; err != nil {
			return err
		}

		// Enabling by owner is blocked if the admin has disabled it.
		if !disabled && m.AdminDisabled {
			return ErrMeterDisabledByAdmin
		}

		relayStatus := desiredRelayStatus(m.AdminDisabled, disabled)
		if err := innerTx.Model(&Meter{}).Where("id = ?", meterID).Updates(map[string]any{
			"owner_disabled": disabled,
			"relay_status":   relayStatus,
		}).Error; err != nil {
			return fmt.Errorf("update owner meter control: %w", err)
		}
		return nil
	})
}


func TurnOffMeter(tx *gorm.DB, meterID uuid.UUID) error {
	if tx == nil {
		tx = db.DB
	}
	return tx.Model(&Meter{}).Where("id = ?", meterID).Update("relay_status", "OFF").Error
}

func TurnOnMeter(tx *gorm.DB, meterID uuid.UUID) error {
	if tx == nil {
		tx = db.DB
	}
	return tx.Model(&Meter{}).Where("id = ?", meterID).Update("relay_status", "ON").Error
}
