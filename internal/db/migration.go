package db

import "log"

func Migrate(models ...any) {
	if err := DB.AutoMigrate(models...); err != nil {
		log.Fatal("Failed to run database migrations:", err)
	}
	log.Println("Database migrations completed successfully")
}
