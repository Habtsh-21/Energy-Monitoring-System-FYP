package models

import "energy-monitoring-system/internal/db"



type Address struct{
	Region string
	City string
	SubCity string
	Kebele string
	HouseNumber string
}


type User struct {
	ID          uint `gorm:"primaryKey"`
	FullName    string
	PhoneNumber string
	Password    string
	Address     Address `gorm:"embedded"`
	MeterNumber string `gorm:"uniqueIndex"`
}


func(user *User) Create() error {
	if err := db.DB.Create(user).Error; err != nil {
		return err
	}
	return nil
}
