package handlers

import (
	"encoding/json"
	"energy-monitoring-system/internal/auth"
	"energy-monitoring-system/internal/db"
	"energy-monitoring-system/internal/models"
	"energy-monitoring-system/internal/utils"
	"net/http"

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

    user, err := models.GetUserByPhone(db.DB, req.PhoneNumber)
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



