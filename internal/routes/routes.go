package routes

import (
"github.com/gorilla/mux"
"energy-monitoring-system/internal/handlers"
"energy-monitoring-system/internal/auth/middleware"
)


func RegisterRoutes(r *mux.Router) {

	user := r.PathPrefix("/user").Subrouter()
	user.Use(middleware.AuthMiddleware)
	user.HandleFunc("/", handlers.UserHomeHandler).Methods("GET")

	admin := r.PathPrefix("/adm").Subrouter()
	admin.Use(middleware.AdminPathPermission)
	admin.HandleFunc("/", handlers.AdminHomeHandler).Methods("GET")
	admin.HandleFunc("/user/register", handlers.UserRegisterHandler).Methods("POST")
	admin.HandleFunc("/user/:id", handlers.GetUserHandler).Methods("GET")
	admin.HandleFunc("/users", handlers.GetAllUserHandler).Methods("GET")
	admin.HandleFunc("/user/:id", handlers.UpdateUserHandler).Methods("PUT")
	admin.HandleFunc("/user/:id", handlers.DeleteUserHandler).Methods("DELETE")
	admin.HandleFunc("/meter/register", handlers.MeterRegisterHandler).Methods("POST")
	admin.HandleFunc("/meter/:id", handlers.GetMeterHandler).Methods("GET")
	admin.HandleFunc("/meters", handlers.GetAllMeterHandler).Methods("GET")
	admin.HandleFunc("/meter/:id", handlers.UpdateMeterHandler).Methods("PUT")
	admin.HandleFunc("/meter/:id", handlers.DeleteMeterHandler).Methods("DELETE")
}