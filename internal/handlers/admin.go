package handlers

import (
	"encoding/json"
	"energy-monitoring-system/internal/db"
	"energy-monitoring-system/internal/models"
	"energy-monitoring-system/internal/utils"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func AdminHomeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Welcome to Energy Monitoring System"))
}

func UserRegisterHandler(w http.ResponseWriter, r *http.Request) {

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

	user.Password, err = utils.HashPassword(user.Password)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	available, err := models.CheckAvailability(user.MeterID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Meter does not exist", http.StatusBadRequest)
			return
		}
		http.Error(w, "Error checking meter availability: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if !available {
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

		if err = models.UpdateMeterStatus(tx, user.MeterID, false); err != nil {
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

func GetDeletedUserHandler(w http.ResponseWriter, r *http.Request) {

	users, err := models.GetAllUserWithDeleted()
	if err != nil {
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func GetAllRecordHandler(w http.ResponseWriter, r *http.Request) {

	records, err := models.GetAllRecord()
	if err != nil {
		http.Error(w, "Failed to get records", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

func GetAllUserHandler(w http.ResponseWriter, r *http.Request) {

	users, err := models.GetAllUser()
	if err != nil {
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func GetUserHandler(w http.ResponseWriter, r *http.Request) {

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

func UpdateUserHandler(w http.ResponseWriter, r *http.Request) {

	var user models.User

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := user.Update(); err != nil {
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func DeleteUserHandler(w http.ResponseWriter, r *http.Request) {

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

	var currentAssignments int64
	if err := db.DB.Model(&models.Record{}).
		Where("user_id = ? AND is_current = ?", user.ID, true).
		Count(&currentAssignments).Error; err != nil {
		http.Error(w, "Failed to check user assignments", http.StatusInternalServerError)
		return
	}
	if currentAssignments > 0 {
		http.Error(w, "User has an active meter assignment. Unassign meter first.", http.StatusConflict)
		return
	}

	if err := user.Delete(); err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func PermanentDeleteUserHandler(w http.ResponseWriter, r *http.Request) {

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

func PermanentDeleteAllUserHandler(w http.ResponseWriter, r *http.Request) {

	if err := models.PermanentUsersDelete(); err != nil {
		http.Error(w, "Failed to delete users", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func MeterRegisterHandler(w http.ResponseWriter, r *http.Request) {

	var meter models.Meter

	if err := json.NewDecoder(r.Body).Decode(&meter); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if meter.MeterSerialNumber == "" || meter.MeterType == "" || meter.Manufacturer == "" {
		http.Error(w, "Incomplete meter information", http.StatusBadRequest)
		return
	}

	meter.ID = utils.IdGenerator()
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

func GetAllMeterHandler(w http.ResponseWriter, r *http.Request) {

	meters, err := models.GetAllMeter()
	if err != nil {
		http.Error(w, "Failed to get meters", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meters)
}

func GetMeterHandler(w http.ResponseWriter, r *http.Request) {

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

func GetDeletedMeterHandler(w http.ResponseWriter, r *http.Request) {

	meters, err := models.GetAllMeterWithDeleted()
	if err != nil {
		http.Error(w, "Failed to get meters", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meters)
}

func UpdateMeterHandler(w http.ResponseWriter, r *http.Request) {

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

func DeleteMeterHandler(w http.ResponseWriter, r *http.Request) {

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

	var assignedUsers int64
	if err := db.DB.Model(&models.User{}).
		Where("meter_id = ?", meter.MeterSerialNumber).
		Count(&assignedUsers).Error; err != nil {
		http.Error(w, "Failed to check meter assignments", http.StatusInternalServerError)
		return
	}
	if assignedUsers > 0 {
		http.Error(w, "Meter is assigned to a user. Unassign meter first.", http.StatusConflict)
		return
	}

	var linkedRecords int64
	if err := db.DB.Model(&models.Record{}).
		Where("meter_id = ?", meter.ID).
		Count(&linkedRecords).Error; err != nil {
		http.Error(w, "Failed to check meter records", http.StatusInternalServerError)
		return
	}
	if linkedRecords > 0 {
		http.Error(w, "Meter has assignment history and cannot be deleted.", http.StatusConflict)
		return
	}

	if err := meter.Delete(); err != nil {
		http.Error(w, "Failed to delete meter", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func PermanentDeleteMeterHandler(w http.ResponseWriter, r *http.Request) {

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

func PermanentDeleteAllMeterHandler(w http.ResponseWriter, r *http.Request) {

	if err := models.PermanentMetersDelete(); err != nil {
		http.Error(w, "Failed to delete meters", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
