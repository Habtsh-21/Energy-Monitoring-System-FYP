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

	admin := r.PathPrefix("/adm").Subrouter()
	admin.Use(middleware.AdminPathPermission)
	admin.HandleFunc("/", handlers.AdminHomeHandler).Methods("GET")
	admin.HandleFunc("/user", handlers.UserRegisterHandler).Methods("POST")
	admin.HandleFunc("/user/{id}", handlers.GetUserHandler).Methods("GET")
	admin.HandleFunc("/users", handlers.GetAllUserHandler).Methods("GET")
	admin.HandleFunc("/user/{id}", handlers.UpdateUserHandler).Methods("PUT")
	admin.HandleFunc("/user/{id}", handlers.DeleteUserHandler).Methods("DELETE")
	admin.HandleFunc("/meter", handlers.MeterRegisterHandler).Methods("POST")
	admin.HandleFunc("/meter/{serialNo}", handlers.GetMeterHandler).Methods("GET")
	admin.HandleFunc("/meters", handlers.GetAllMeterHandler).Methods("GET")
	admin.HandleFunc("/meter/{serialNo}", handlers.UpdateMeterHandler).Methods("PUT")
	admin.HandleFunc("/meter/{serialNo}", handlers.DeleteMeterHandler).Methods("DELETE")
}
