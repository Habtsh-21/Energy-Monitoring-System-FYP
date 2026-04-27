package handlers

import (
	"encoding/json"
	"energy-monitoring-system/internal/db"
	"energy-monitoring-system/internal/models"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminHandler struct{}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

func (h *AdminHandler) AdminHomeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Welcome to Energy Monitoring System"))
}

func (h *AdminHandler) UserRegisterHandler(w http.ResponseWriter, r *http.Request) {

	var user models.User
	var record models.Record
	var err error

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if user.FullName == "" || user.PhoneNumber == "" || user.Password == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	if len(user.Password) < 6 {
		http.Error(w, "Password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	user.Password = string(hashedPassword)

	isAvailable, err := models.IsMeterAvailable(user.MeterID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Meter does not exist", http.StatusBadRequest)
			return
		}
		http.Error(w, "Error checking meter availability: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if !isAvailable {
		http.Error(w, "Meter is not available for assignment", http.StatusBadRequest)
		return
	}

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.IsActive = true

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := user.Create(tx); err != nil {
			return err
		}

		record = models.Record{
			UserID:     user.ID,
			MeterID:    user.MeterID,
			AssignedAt: time.Now(),
			IsCurrent:  true,
			AssignedBy: "admin",
		}
		record.BaseModel.CreatedAt = time.Now()
		record.BaseModel.UpdatedAt = time.Now()

		if err := record.Create(tx); err != nil {
			return err
		}

		if err = models.UpdateMeterParameters(tx, user.MeterID, map[string]any{"is_available": false}); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		http.Error(w, "Failed to register: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *AdminHandler) GetAllUserHandler(w http.ResponseWriter, r *http.Request) {

	users, err := models.GetAllUser()
	if err != nil {
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (h *AdminHandler) GetUserHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	userId, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid or missing user ID", http.StatusBadRequest)
		return
	}

	user, err := models.GetUser(userId)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *AdminHandler) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userId, err := uuid.Parse(vars["userId"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	updates["updated_at"] = time.Now()
	if len(updates) == 0 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	if err := models.UpdateUserParameters(db.DB, userId, updates); err != nil {
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AdminHandler) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	userId, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid or missing user ID", http.StatusBadRequest)
		return
	}
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		user, err := models.GetUser(userId)
		if err != nil {
			return err
		}

		if err := models.UpdateUserParameters(tx, userId, map[string]any{"is_active": false}); err != nil {
			return err
		}

		if err := models.UpdateMeterParameters(tx, user.MeterID, map[string]any{"is_available": true}); err != nil {
			return err
		}
		if err := user.Delete(tx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		http.Error(w, "Failed to process request", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AdminHandler) PermanentDeleteUserHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	userId, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid or missing user ID", http.StatusBadRequest)
		return
	}

	if err := models.PermanentUserDelete(userId); err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AdminHandler) MeterRegisterHandler(w http.ResponseWriter, r *http.Request) {

	var meter models.Meter

	if err := json.NewDecoder(r.Body).Decode(&meter); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if meter.MeterSerialNumber == "" || meter.MeterType == "" {
		http.Error(w, "Incomplete meter information", http.StatusBadRequest)
		return
	}
	meter.CreatedAt = time.Now()
	meter.UpdatedAt = time.Now()

	if models.CheckSerialNo(meter.MeterSerialNumber) {
		http.Error(w, "Meter serial number already exists", http.StatusBadRequest)
		return
	}

	if err := meter.Create(); err != nil {
		http.Error(w, "Failed to create meter", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AdminHandler) GetAllMeterHandler(w http.ResponseWriter, r *http.Request) {

	meters, err := models.GetAllMeter()
	if err != nil {
		http.Error(w, "Failed to get meters", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meters)
}

func (h *AdminHandler) GetMeterHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	meterID := vars["id"]
	if meterID == "" {
		http.Error(w, "Meter ID is required", http.StatusBadRequest)
		return
	}

	meter, err := models.GetMeterByID(uuid.Must(uuid.Parse(meterID)))
	if err != nil {
		http.Error(w, "Failed to get meter", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meter)
}

func (h *AdminHandler) GetDeletedMeterHandler(w http.ResponseWriter, r *http.Request) {

	meters, err := models.GetAllMeterWithDeleted()
	if err != nil {
		http.Error(w, "Failed to get meters", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meters)
}

func (h *AdminHandler) UpdateMeterHandler(w http.ResponseWriter, r *http.Request) {

	var meter models.Meter

	if err := json.NewDecoder(r.Body).Decode(&meter); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := meter.Update(); err != nil {
		http.Error(w, "Failed to update meter", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AdminHandler) DeleteMeterHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	meterID := vars["id"]
	if meterID == "" {
		http.Error(w, "Meter ID is required", http.StatusBadRequest)
		return
	}

	meter, err := models.GetMeterByID(uuid.Must(uuid.Parse(meterID)))
	if err != nil {
		http.Error(w, "Failed to get meter", http.StatusInternalServerError)
		return
	}

	if err := meter.Delete(); err != nil {
		http.Error(w, "Failed to delete meter", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AdminHandler) PermanentDeleteMeterHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	meterID := vars["id"]
	if meterID == "" {
		http.Error(w, "Meter ID is required", http.StatusBadRequest)
		return
	}

	meter, err := models.GetMeterByID(uuid.Must(uuid.Parse(meterID)))
	if err != nil {
		http.Error(w, "Failed to get meter", http.StatusInternalServerError)
		return
	}

	if err := meter.Delete(); err != nil {
		http.Error(w, "Failed to delete meter", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AdminHandler) PermanentDeleteAllMeterHandler(w http.ResponseWriter, r *http.Request) {

	if err := models.PermanentMetersDelete(); err != nil {
		http.Error(w, "Failed to delete meters", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AdminHandler) GetllRecordHandler(w http.ResponseWriter, r *http.Request) {

	records, err := models.GetAllRecord()
	if err != nil {
		http.Error(w, "Failed to get records", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

type ResolveAnomalyRequest struct {
	ResolvedBy     string `json:"resolved_by"`
	ResolutionNote string `json:"resolution_note"`
}

func (h *AdminHandler) GetAnomaliesHandler(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && status != models.AnomalyStatusOpen && status != models.AnomalyStatusResolved {
		http.Error(w, "status must be open or resolved", http.StatusBadRequest)
		return
	}

	anomalies, err := models.GetAnomalies(status)
	if err != nil {
		http.Error(w, "Failed to get anomalies", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(anomalies)
}

func (h *AdminHandler) ResolveAnomalyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	anomalyID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid anomaly ID", http.StatusBadRequest)
		return
	}

	var req ResolveAnomalyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resolvedBy := strings.TrimSpace(req.ResolvedBy)
	if resolvedBy == "" {
		resolvedBy = "admin"
	}

	if err := models.ResolveAnomaly(anomalyID, resolvedBy, strings.TrimSpace(req.ResolutionNote)); err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Anomaly not found or already resolved", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to resolve anomaly", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
