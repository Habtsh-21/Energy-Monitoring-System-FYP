package routes

import (
	"energy-monitoring-system/internal/auth/middleware"
	"energy-monitoring-system/internal/handlers"

	"github.com/gorilla/mux"
)

func RegisterRoutes(r *mux.Router) {
	
	user := r.PathPrefix("/user").Subrouter()
	user.Use(middleware.AuthMiddleware)
	user.HandleFunc("/", handlers.UserHomeHandler).Methods("GET")
	r.HandleFunc("/login", handlers.LoginHandler).Methods("POST")

	adminHandler := handlers.NewAdminHandler()
	admin := r.PathPrefix("/adm").Subrouter()
	admin.Use(middleware.AdminPathPermission)
	admin.HandleFunc("/", adminHandler.AdminHomeHandler).Methods("GET")
	admin.HandleFunc("/user", adminHandler.UserRegisterHandler).Methods("POST")
	admin.HandleFunc("/user/{id}", adminHandler.GetUserHandler).Methods("GET")
	admin.HandleFunc("/users", adminHandler.GetAllUserHandler).Methods("GET")
	admin.HandleFunc("/user/{id}", adminHandler.UpdateUserHandler).Methods("PUT")
	admin.HandleFunc("/user/{id}", adminHandler.DeleteUserHandler).Methods("DELETE")
	admin.HandleFunc("/meter", adminHandler.MeterRegisterHandler).Methods("POST")
	admin.HandleFunc("/meter/{id}", adminHandler.GetMeterHandler).Methods("GET")
	admin.HandleFunc("/meter/{id}", adminHandler.UpdateMeterHandler).Methods("PUT")
	admin.HandleFunc("/meters", adminHandler.GetAllMeterHandler).Methods("GET")
	admin.HandleFunc("/meter/{id}", adminHandler.DeleteMeterHandler).Methods("DELETE")
	admin.HandleFunc("/records", adminHandler.GetllRecordHandler).Methods("GET")

	r.HandleFunc("/reading", handlers.MeterReadingHandler).Methods("POST")
	r.HandleFunc("/reading/meter/{id}", handlers.GetMeterReadingHandler).Methods("GET")
	r.HandleFunc("/reading/user/{id}", handlers.GetUserReadingHandler).Methods("GET")


}
