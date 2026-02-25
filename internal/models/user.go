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

func(user *User) Update() error {
	if err := db.DB.Save(user).Error; err != nil {
		return err
	}
	return nil
}

func(user *User) Delete() error {
	if err := db.DB.Delete(user).Error; err != nil {
		return err
	}
	return nil
}

func GetUser(userId string) (*User, error) {
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