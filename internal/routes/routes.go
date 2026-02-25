package routes

import (
"github.com/gorilla/mux"
"energy-monitoring-system/internal/handlers"
"energy-monitoring-system/internal/auth/middleware"
)


func RegisterRoutes(r *mux.Router) {

	user := r.PathPrefix("/user").Subrouter()
	user.Use(middleware.AuthMiddleware)
	user.HandleFunc("/", handlers.HomeHandler).Methods("GET")

	admin := r.PathPrefix("/admin").Subrouter()
	admin.Use(middleware.AdminPathPermission)
	admin.HandleFunc("/", handlers.HomeHandler).Methods("GET")
}