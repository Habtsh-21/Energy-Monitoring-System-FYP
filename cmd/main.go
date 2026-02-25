package main

import (
	"log"
	"net/http"
	"os"
	"energy-monitoring-system/internal/db"
	"energy-monitoring-system/internal/models"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)


func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}
	log.Println("Environment variables loaded successfully")

	db.InitDB()

	if err := db.DB.AutoMigrate(&models.User{}, &models.Meter{}); err != nil {
		log.Fatal("Failed to migrate models:", err)
	}

	r := mux.NewRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
