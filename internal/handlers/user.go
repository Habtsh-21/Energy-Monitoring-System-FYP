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


func UserHomeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Welcome to Energy Monitoring System"))
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
