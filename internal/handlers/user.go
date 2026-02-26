package handlers

import (
	"encoding/json"
	"energy-monitoring-system/internal/auth"
	"energy-monitoring-system/internal/db"
	"energy-monitoring-system/internal/models"
	"energy-monitoring-system/internal/utils"
	"net/http"
)

func UserHomeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Welcome to Energy Monitoring System"))
}


func LoginHandler(w http.ResponseWriter, r *http.Request) {

	var user models.User

	var creds struct {
		UserId   string `json:"user_id"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := db.DB.Where("ID = ?", creds.UserId).First(&user).Error; err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := utils.VerifyPassword(creds.Password, user.PasswordHash); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateJWT(&user)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})

}
