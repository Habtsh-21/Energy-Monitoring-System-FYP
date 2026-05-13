package routes

import (
	"energy-monitoring-system/internal/auth/middleware"
	"energy-monitoring-system/internal/handlers"
	"energy-monitoring-system/internal/handlers/admin_mgt"

	"github.com/gorilla/mux"
)

func RegisterRoutes(r *mux.Router) {

	user := r.PathPrefix("/user").Subrouter()
	user.Use(middleware.AuthMiddleware())
	user.HandleFunc("/", handlers.UserHomeHandler).Methods("GET")
	r.HandleFunc("/login", handlers.LoginHandler).Methods("POST")

	user.HandleFunc("/wallet/topup", handlers.WalletTopUpHandler).Methods("POST")
	user.HandleFunc("/wallet/balance", handlers.GetWalletBalanceHandler).Methods("GET")
	user.HandleFunc("/wallet/transactions", handlers.GetWalletTransactionsHandler).Methods("GET")
	user.HandleFunc("/reports", handlers.CreateReportHandler).Methods("POST")
	// user.HandleFunc("/reports", handlers.GetMyReportsHandler).Methods("GET")
	// user.HandleFunc("/reports/{id}", handlers.GetMyReportByIDHandler).Methods("GET")

	adminHandler := handlers.NewAdminHandler()
	admin := r.PathPrefix("/admin").Subrouter() 
	admin.Use(middleware.AuthMiddleware())
	r.HandleFunc("/admin/login", handlers.AdminLoginHandler).Methods("POST")
	admin.HandleFunc("/dashboard", adminHandler.DashboardHandler).Methods("GET")
	admin.HandleFunc("/", adminHandler.AdminHomeHandler).Methods("GET")
	admin.HandleFunc("/user", admin_mgt.UserRegisterHandler).Methods("POST")
	admin.HandleFunc("/user/{id}", admin_mgt.GetUserHandler).Methods("GET")
	admin.HandleFunc("/user/phone/{phoneNumber}", admin_mgt.GetUserByPhoneHandler).Methods("GET")
	admin.HandleFunc("/users", admin_mgt.GetAllUserHandler).Methods("GET")
	admin.HandleFunc("/user/{id}", admin_mgt.UpdateUserHandler).Methods("PUT")
	admin.HandleFunc("/user/{id}", admin_mgt.DeleteUserHandler).Methods("DELETE")
	admin.HandleFunc("/meter", admin_mgt.MeterRegisterHandler).Methods("POST")
	admin.HandleFunc("/meter/{id}", admin_mgt.GetMeterHandler).Methods("GET")
	admin.HandleFunc("/meter/{id}", admin_mgt.UpdateMeterHandler).Methods("PUT")
	admin.HandleFunc("/meters", admin_mgt.GetAllMeterHandler).Methods("GET")
	admin.HandleFunc("/meter/{id}", admin_mgt.DeleteMeterHandler).Methods("DELETE")
	admin.HandleFunc("/user/{id}/meter", admin_mgt.ChangeMeterHandler).Methods("PUT")
	admin.HandleFunc("/records", admin_mgt.GetAllRecordHandler).Methods("GET")
	admin.HandleFunc("/anomalies", adminHandler.GetAnomaliesHandler).Methods("GET")
	admin.HandleFunc("/anomaly/{id}", adminHandler.GetAnomalyDetailHandler).Methods("GET")
	admin.HandleFunc("/reports", handlers.GetAllReportsHandler).Methods("GET")
	admin.HandleFunc("/reports/{id}", handlers.GetReportByIDHandler).Methods("GET")
	admin.HandleFunc("/reports/{id}/status", handlers.UpdateReportStatusHandler).Methods("PATCH")
	admin.HandleFunc("/transactions", handlers.GetAllTransactionHandler).Methods("GET")
	// r.HandleFunc("/tariffs", handlers.AdminSetTariffHandler).Methods("POST")
	// r.HandleFunc("/tariffs", handlers.AdminGetTariffsHandler).Methods("GET")

	admin.HandleFunc("/line-reading", handlers.LineReadingHandler).Methods("POST")
	admin.HandleFunc("/line-reading/{id}", handlers.GetReadingRecord).Methods("GET")
	admin.HandleFunc("/line-reading", handlers.GetMetersReadings).Methods("GET")
	// r.HandleFunc("/calculate-cost", handlers.CalculateCostHandler).Methods("GET")
	// r.HandleFunc("/calculate-kwh", handlers.CalculateKwhHandler).Methods("GET")
}
