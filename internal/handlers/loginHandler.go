package handlers

import (
	"encoding/json"
	"energy-monitoring-system/internal/auth"
	"energy-monitoring-system/internal/db"
	"fmt"
	"net/http"

	"gorm.io/gorm"
)

func Login(dbConn *gorm.DB, userId string, password string) (string, error) {

	var user db.User
	if err := dbConn.Where("ID = ?", userId).First(&user).Error; err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	if err := auth.VerifyPassword(password, user.Password); err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	token, err := auth.GenerateJWT(&user)
	if err != nil {
		return "", err
	}

	return token, nil
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		UserId   string `json:"user_id"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	token, err := Login(db.DB, creds.UserId, creds.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}
