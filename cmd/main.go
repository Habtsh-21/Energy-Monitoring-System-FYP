package main

import (
	"log"
	"net/http"
	"os"

	"energy-monitoring-system/internal/auth"
	"energy-monitoring-system/internal/db"
	"energy-monitoring-system/internal/handlers"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Welcome to Energy Monitoring System"))
}

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}
	log.Println("Environment variables loaded successfully")

	db.InitDB()

	r := mux.NewRouter()

	r.HandleFunc("/", homeHandler).Methods("GET")
	r.HandleFunc("/login", handlers.LoginHandler).Methods("POST")

	api := r.PathPrefix("/api").Subrouter()
	api.Use(auth.AuthMiddleware)

	api.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(auth.UserIDKey)
		w.Write([]byte("Hello User " + userID.(string)))
	}).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
