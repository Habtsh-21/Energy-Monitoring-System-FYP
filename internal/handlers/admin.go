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

	if user.FullName == "" || user.PhoneNumber == "" || user.PasswordHash == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}
	if len(user.PasswordHash) < 6 {
		http.Error(w, "Password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	if models.CheckPhoneNumber(user.PhoneNumber) {
		http.Error(w, "Phone number already exists", http.StatusBadRequest)
		return
	}
	if models.CheckMeterSerialNo(user.AssignedMeterSerialNo) {
		http.Error(w, "Meter serial number already Assigned", http.StatusBadRequest)
		return
	}

	user.ID = utils.IdGenerator()

	for models.CheckUserId(user.ID) {
		user.ID = utils.IdGenerator()
	}

	user.PasswordHash, err = utils.HashPassword(user.PasswordHash)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.IsActive = true

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := user.Create(tx); err != nil {
			return err
		}

		meterID, err := models.GetMeterID(user.AssignedMeterSerialNo)
		if err != nil {
			return err
		}

		record = models.Record{
			ID:         utils.IdGenerator(),
			UserID:     user.ID,
			MeterID:    meterID,
			AssignedAt: time.Now(),
			IsCurrent:  true,
			AssignedBy: "admin",
		}
		record.BaseModel.CreatedAt = time.Now()
		record.BaseModel.UpdatedAt = time.Now()

		if err := record.Create(tx); err != nil {
			return err
		}
         
		if err = models.UpdateMeterStatus( tx ,meterID, "assigned"); err != nil {
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

	if err := user.Delete(); err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
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

	meter.ID = utils.IdGenerator()
	meter.CreatedAt = time.Now()

	if models.CheckMeterSerialNo(meter.MeterSerialNumber) {
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
	serialNo := vars["serialNo"]
	if serialNo == "" {
		http.Error(w, "Meter ID is required", http.StatusBadRequest)
		return
	}

	meter, err := models.GetMeter(serialNo)
	if err != nil {
		http.Error(w, "Failed to get meter", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meter)
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
	serialNo := vars["serialNo"]
	if serialNo == "" {
		http.Error(w, "Meter ID is required", http.StatusBadRequest)
		return
	}

	meter, err := models.GetMeter(serialNo)
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
