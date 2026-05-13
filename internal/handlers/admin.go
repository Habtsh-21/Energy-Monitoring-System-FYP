package handlers

import (
	"encoding/json"
	"energy-monitoring-system/internal/auth"
	"energy-monitoring-system/internal/db"
	"energy-monitoring-system/internal/models"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type AdminHandler struct{}

//Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NzkxOTY5MzUsImlzcyI6ImVuZXJneS1tb25pdG9yaW5nLXN5c3RlbSIsInN1YiI6ImFjY2VzcyIsInVzZXJfaWQiOiIwMDAwMDAwMC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDAifQ.XY-JP3z-I8UNmkIC3fOzuU5LvnmjcTzGisTuPdzN6iU

type AdminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

func (h *AdminHandler) AdminHomeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Welcome to Energy Monitoring System"))
}

func AdminLoginHandler(w http.ResponseWriter, r *http.Request) {
	var req AdminLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	adminUsername := os.Getenv("ADMIN_USERNAME")
	adminPassword := os.Getenv("PASSWORD")
	if adminUsername == "" || adminPassword == "" {
		http.Error(w, "admin credentials are not configured", http.StatusInternalServerError)
		return
	}

	if req.Username != adminUsername || req.Password != adminPassword {
		http.Error(w, "Invalid credentials. c_u:"+req.Username+" c_p:"+req.Password+" a_u:"+adminUsername+" a_p:"+adminPassword, http.StatusUnauthorized)
		return
	}

	adminID := uuid.Nil
	if adminIDEnv := os.Getenv("ADMIN_ID"); adminIDEnv != "" {
		parsedID, err := uuid.Parse(adminIDEnv)
		if err != nil {
			http.Error(w, "Invalid ADMIN_ID in environment", http.StatusInternalServerError)
			return
		}
		adminID = parsedID
	}

	token, err := auth.GenerateJWTWithRole(&models.User{
		BaseModel: models.BaseModel{
			ID: adminID,
		},
	}, "admin")
	if err != nil {
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

type AdminDashboardResponse struct {
	TotalUsers             int64   `json:"total_users"`
	ActiveUsers            int64   `json:"active_users"`
	TotalRegisteredMeters  int64   `json:"total_registered_meters"`
	AssignedMeters         int64   `json:"assigned_meters"`
	AvailableMeters        int64   `json:"available_meters"`
	TotalAnomaliesDetected int64   `json:"total_anomalies_detected"`
	TotalReadings          int64   `json:"total_readings"`
	TotalPowerUsageVA      float64 `json:"total_power_usage_va"`
	AveragePowerLossPct    float64 `json:"average_power_loss_pct"`
}

func (h *AdminHandler) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	var res AdminDashboardResponse

	if err := db.DB.Model(&models.User{}).Count(&res.TotalUsers).Error; err != nil {
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}
	if err := db.DB.Model(&models.User{}).Where("is_active = ?", true).Count(&res.ActiveUsers).Error; err != nil {
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}
	if err := db.DB.Model(&models.Meter{}).Count(&res.TotalRegisteredMeters).Error; err != nil {
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}
	if err := db.DB.Model(&models.Meter{}).Where("is_available = ?", false).Count(&res.AssignedMeters).Error; err != nil {
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}
	if err := db.DB.Model(&models.Meter{}).Where("is_available = ?", true).Count(&res.AvailableMeters).Error; err != nil {
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}
	if err := db.DB.Model(&models.Anomaly{}).Count(&res.TotalAnomaliesDetected).Error; err != nil {
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}
	if err := db.DB.Model(&models.LineReading{}).Count(&res.TotalReadings).Error; err != nil {
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}
	if err := db.DB.Model(&models.LineReading{}).
		Select("COALESCE(SUM(meter_apparent_power_va), 0)").
		Scan(&res.TotalPowerUsageVA).Error; err != nil {
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}
	if err := db.DB.Model(&models.LineReading{}).
		Select("COALESCE(AVG(power_loss_pct), 0)").
		Scan(&res.AveragePowerLossPct).Error; err != nil {
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *AdminHandler) GetAnomaliesHandler(w http.ResponseWriter, r *http.Request) {
	anomalies, err := models.GetAnomalies()
	if err != nil {
		http.Error(w, "Failed to fetch anomalies", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Anomalies []models.AnomalyResponse `json:"anomalies"`
	}{Anomalies: anomalies})
}

func (h *AdminHandler) GetAnomalyDetailHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
    
	uuid, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	anomaly, err := models.GetAnomalyByID(uuid)
	if err != nil {
		http.Error(w, "Failed to fetch anomaly", http.StatusInternalServerError)
		return
	}
	

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(anomaly)
}
