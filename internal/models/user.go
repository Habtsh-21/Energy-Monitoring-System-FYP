package models

import (
	"energy-monitoring-system/internal/db"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Address struct {
	Region      string `json:"region"`
	City        string `json:"city"`
	SubCity     string `json:"sub_city"`
	Kebele      string `json:"kebele"`
	HouseNumber string `json:"house_number"`
}
type BaseModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type User struct {
	BaseModel

	Password    string     `gorm:"column:password;not null;default:''" json:"-"`
	FullName    string     `gorm:"column:full_name;size:255;not null" json:"full_name" validate:"required"`
	PhoneNumber string     `gorm:"column:phone_number;size:20;uniqueIndex:idx_phone_number,where:deleted_at IS NULL" json:"phone_number"`
	Address     Address    `gorm:"embedded" json:"address"`
	LastLogin   *time.Time `gorm:"column:last_login" json:"last_login"`
	IsActive    bool       `gorm:"default:true" json:"is_active"`

	MeterID uuid.UUID `gorm:"column:meter_id;type:uuid;uniqueIndex:idx_meter_id,where:deleted_at IS NULL" json:"meter_id"`
    SerialNumber string `gorm:"-" json:"meter_serial_number,omitempty"`	
	Meter       *Meter        `gorm:"foreignKey:MeterID" json:"meter,omitempty"`
	Record      []Record      `gorm:"foreignKey:UserID" json:"records,omitempty"`
	LineReading []LineReading `gorm:"foreignKey:UserID" json:"line_readings,omitempty"`
}



type UserRegisterRequest struct {
    Password    string     `json:"password"`
	FullName    string     `gorm:"column:full_name;size:255;not null" json:"full_name" validate:"required"`
	PhoneNumber string     `gorm:"column:phone_number;size:20;uniqueIndex:idx_phone_number,where:deleted_at IS NULL" json:"phone_number"`
	Address     Address    `gorm:"embedded" json:"address"`
	SerialNumber string `gorm:"-" json:"meter_serial_number,omitempty"`

}

type UsersGetResponse struct {
	ID        uuid.UUID      `json:"id"`
	FullName    string     `json:"full_name" `
	PhoneNumber string     `json:"phone_number"`
	Address     Address    `gorm:"embedded" json:"address"`
	LastLogin   *time.Time `json:"last_login"`
	IsActive    bool       `json:"is_active"`
	MeterID uuid.UUID `json:"meter_id"`
	MeterSerialNumber string `gorm:"-" json:"meter_serial_number,omitempty"`
}

func (user *User) Create(tx *gorm.DB) error {
	if tx == nil {
		tx = db.DB
	}
	if err := tx.Omit(clause.Associations).Create(user).Error; err != nil {
		return err
	}
	return nil
}

func (user *User) Update() error {
	if err := db.DB.Save(user).Error; err != nil {
		return err
	}
	return nil
}

func UpdateUserParameters(tx *gorm.DB, userId uuid.UUID, updates map[string]any) error {
	if err := tx.Model(&User{}).Where("id = ?", userId).Updates(updates).Error; err != nil {
		return err
	}
	return nil
}

func (user *User) Delete(tx *gorm.DB) error {
	if tx == nil {
		tx = db.DB
	}
	if err := tx.Delete(user).Error; err != nil {
		return err
	}
	return nil
}

func GetUser(userId uuid.UUID) (*User, error) {
	var user User

	if err := db.DB.
		Preload("Meter").
		Preload("Record").
		Preload("LineReading").
		Where("id = ?", userId).
		First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func GetUserByPhone(phone string) (*User, error) {
	var user User
	err := db.DB.Where("phone_number = ?", phone).First(&user).Error
	return &user, err
}  


func GetUserIdByMeterId(meterId uuid.UUID) (uuid.UUID, error) {
	var user User
	if err := db.DB.Where("meter_id = ? AND is_active = ?", meterId, true).First(&user).Error; err != nil {
		return uuid.Nil, err
	}
	return user.ID, nil
}



func GetAllUser() ([]User, error) {
	var users []User
	if err := db.DB.Find(&users).Error; err != nil {
		return nil, err
	} 
	return users, nil
}



func CheckPhoneNumber(phoneNumber string) bool {
	var user User
	if err := db.DB.Where("phone_number = ?", phoneNumber).First(&user).Error; err != nil {
		return false
	}
	return true
}


func CheckUserId(id uuid.UUID) bool {
	var user User
	if err := db.DB.Where("id = ?", id).First(&user).Error; err != nil {
		return false
	}
	return true
}

func CheckMeterAssignment(meterID string) bool {
	var user User
	if err := db.DB.Where("meter_id = ?", meterID).First(&user).Error; err != nil {
		return false
	}
	return true
}


 
