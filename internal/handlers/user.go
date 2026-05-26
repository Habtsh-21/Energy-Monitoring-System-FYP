package handlers

import (
	"encoding/json"
	"energy-monitoring-system/internal/auth"
	"energy-monitoring-system/internal/auth/middleware"
	"energy-monitoring-system/internal/models"
	"energy-monitoring-system/internal/utils"
	"net/http"

	"github.com/google/uuid"
)

type LoginRequest struct {
    PhoneNumber string `json:"phone_number" validate:"required"`
    Password    string `json:"password" validate:"required"`
}


func UserInfoHandler(w http.ResponseWriter, r *http.Request) {

    userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	reading, err := models.GetUser(userID);
	if(err != nil) {
		http.Error(w, "Failed to fetch reading", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(reading)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
    var req LoginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    user, err := models.GetUserByPhone(req.PhoneNumber)
    if err != nil {
        http.Error(w, "Invalid credentials" + err.Error(), http.StatusUnauthorized)
        return
    }
    
	err = utils.VerifyPassword(req.Password, user.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

    token, err := auth.GenerateJWT(user)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func OwnerControlMeterHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := models.GetUser(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if user.MeterID == uuid.Nil {
		http.Error(w, "No meter assigned to user", http.StatusBadRequest)
		return
	}
	
	meter, err  := models.GetMeterByID(user.MeterID)
	if err != nil {
		http.Error(w, "Meter not found", http.StatusNotFound)
		return
	}


	var req struct {
		Disabled bool `json:"disabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !req.Disabled && !user.IsActive {
		http.Error(w, "Cannot enable meter: user account is inactive", http.StatusForbidden)
		return
	}

	
	if meter.AdminDisabled && !req.Disabled {
		http.Error(w, "Cannot enable meter: admin desabled the meter", http.StatusForbidden)
		return
	}

	if err := models.SetOwnerDisabled(nil, user.MeterID, req.Disabled); err != nil {
		
		http.Error(w, "Failed to update meter control: "+err.Error(), http.StatusInternalServerError)
		return
	}

	msg := "Meter enabled by owner"
	if req.Disabled {
		msg = "Meter disabled by owner — relay forced OFF"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": msg})
}
