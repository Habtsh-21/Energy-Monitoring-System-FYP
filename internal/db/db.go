package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

type User struct {
	ID          uint `gorm:"primaryKey"`
	FullName    string
	PhoneNumber string
	Password    string
	Address     string
	MeterNumber string `gorm:"uniqueIndex"`
}

type Meter struct {
	ID          uint   `gorm:"primaryKey"`
	MeterNumber string `gorm:"uniqueIndex"`
	
}

type Reading struct {
	ID                     uint   `gorm:"primaryKey"`
	MeterNumber            string `gorm:"uniqueIndex"`
	Energy_At_Pole_kwh     float64
	Energy_At_Consumer_kwh float64
	Timestamp              time.Time
}

type UtilityCenter struct {
	ID          uint   `gorm:"primaryKey"`
	UtilityID   string `gorm:"uniqueIndex"`
	Name        string
	Region      string
	City        string
	Address     string
	ContactInfo string
}

func InitDB() {
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host, user, password, dbname, port)

	var err error
	for i := 0; i < 10; i++ {
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("Connecting to DB... (attempt %d/10): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatal("Could not connect to database after 10 attempts:", err)
	}

	log.Println("Database connection established successfully")

	err = DB.AutoMigrate(&User{}, &Meter{}, &Reading{}, &UtilityCenter{})
	if err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	log.Println("Database migrations completed successfully")
}
