package admin_mgt

import (
	"encoding/json"
	"energy-monitoring-system/internal/db"
	"energy-monitoring-system/internal/models"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)




  



func  UserRegisterHandler(w http.ResponseWriter, r *http.Request) {

	var user models.User
	var record models.Record
	var err error

	var req models.UserRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Println("--------------", err)
		http.Error(w, "Invalid request body"+err.Error(), http.StatusBadRequest)
		return
	}

	user.FullName = req.FullName
	user.PhoneNumber = req.PhoneNumber
	user.Password = req.Password
	user.Address = req.Address
	user.SerialNumber = req.SerialNumber
     fmt.Println("req:",req)
	id, err := models.GetMeterIDBySerialNo(user.SerialNumber)
	if err != nil {
		http.Error(w, "Failed to find meter"+err.Error(), http.StatusInternalServerError)
		return
	}
	user.MeterID = id
	
	if user.FullName == "" || user.PhoneNumber == "" || user.Password == "" || user.SerialNumber == "" {
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
    isphonenumberavailable:=models.CheckPhoneNumber(user.PhoneNumber)
	if isphonenumberavailable {
		http.Error(w, "Phone number is not available for assignment", http.StatusBadRequest)
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
		//create wallet for a user
		wallet := models.Wallet{
			UserID: user.ID,
		}
		if err := wallet.Create(tx); err != nil {
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


func GetUserByPhoneHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	phoneNumber := vars["phoneNumber"]
	if phoneNumber == "" {
		http.Error(w, "Invalid or missing phone number", http.StatusBadRequest)
		return
	}

	user, err := models.GetUserByPhone(phoneNumber)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}



func UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
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


func ChangeMeterHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid or missing user ID", http.StatusBadRequest)
		return
	}

	var req struct {
		MeterID uuid.UUID `json:"meter_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.MeterID == uuid.Nil {
		http.Error(w, "meter_id is required", http.StatusBadRequest)
		return
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		user, err := models.GetUser(userID)
		if err != nil {
			return err
		}

		if user.MeterID == req.MeterID {
			return nil
		}

		isAvailable, err := models.IsMeterAvailable(req.MeterID)
		if err != nil {
			return err
		}
		if !isAvailable {
			return gorm.ErrInvalidData
		}

		now := time.Now()
		if err := tx.Model(&models.Record{}).
			Where("user_id = ? AND is_current = ?", userID, true).
			Updates(map[string]any{
				"is_current":         false,
				"unassigned_at":      now,
				"termination_reason": "meter changed",
				"updated_at":         now,
			}).Error; err != nil {
			return err
		}

		newRecord := models.Record{
			UserID:     userID,
			MeterID:    req.MeterID,
			AssignedAt: now,
			AssignedBy: "admin",
			IsCurrent:  true,
		}
		newRecord.BaseModel.CreatedAt = now
		newRecord.BaseModel.UpdatedAt = now
		if err := newRecord.Create(tx); err != nil {
			return err
		}

		if err := models.UpdateUserParameters(tx, userID, map[string]any{
			"meter_id":   req.MeterID,
			"updated_at": now,
		}); err != nil {
			return err
		}

		if err := models.UpdateMeterParameters(tx, user.MeterID, map[string]any{"is_available": true}); err != nil {
			return err
		}
		if err := models.UpdateMeterParameters(tx, req.MeterID, map[string]any{"is_available": false}); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		if err == gorm.ErrInvalidData {
			http.Error(w, "New meter is not available", http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to change meter", http.StatusInternalServerError)
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
