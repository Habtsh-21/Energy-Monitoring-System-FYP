package models

import (
	"energy-monitoring-system/internal/db"
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Address struct {
	Region      string
	City        string
	SubCity     string
	Kebele      string
	HouseNumber string
}
type BaseModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type User struct {
    BaseModel
    
    PasswordHash  string    `gorm:"column:password_hash;not null" json:"-"`
    FullName      string    `gorm:"column:full_name;size:255;not null" json:"full_name" validate:"required"`
    PhoneNumber   string    `gorm:"column:phone_number;size:20" json:"phone_number"`
    Address       Address `gorm:"embedded" json:"address"`
    LastLogin     *time.Time `gorm:"column:last_login" json:"last_login"`
    IsActive      bool      `gorm:"default:true" json:"is_active"`
	
    CurrentMeterID *uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"current_meter_id"`
    CurrentMeter   *Meter     `gorm:"foreignKey:CurrentMeterID" json:"current_meter,omitempty"`

    Record []Record `gorm:"foreignKey:UserID" json:"meter_assignments,omitempty"`

}


func (user *User) Create() error {
	if err := db.DB.Create(user).Error; err != nil {
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

func (user *User) Delete() error {
	if err := db.DB.Delete(user).Error; err != nil {
		return err
	}
	return nil
}

func GetUser(userId uuid.UUID) (*User, error) {
	var user User
	if err := db.DB.Where("ID = ?", userId).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
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
	if err := db.DB.Where("PhoneNumber = ?", phoneNumber).First(&user).Error; err != nil {
		return false
	}
	return true
}

func CheckUserId(id uuid.UUID) bool {
	var user User
	if err := db.DB.Where("ID = ?", id).First(&user).Error; err != nil {
		return false
	}
	return true
}
