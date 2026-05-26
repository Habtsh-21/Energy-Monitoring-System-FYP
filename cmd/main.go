package main

import (
	"energy-monitoring-system/internal/db"
	"energy-monitoring-system/internal/models"
	"energy-monitoring-system/internal/routes"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}
	log.Println("Environment variables loaded successfully")

	db.InitDB()
	db.Migrate(
		&models.User{},
		&models.Meter{},
		&models.Record{},
		&models.LineReading{},
		&models.Anomaly{},
		&models.Report{},
		&models.Wallet{},
		&models.Transaction{},
		&models.TariffTier{},
	)

	r := mux.NewRouter()
	routes.RegisterRoutes(r)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, r))
}
 