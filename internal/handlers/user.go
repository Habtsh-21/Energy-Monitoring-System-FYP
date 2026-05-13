package handlers

import (
	"encoding/json"
	"energy-monitoring-system/internal/auth"
	"energy-monitoring-system/internal/models"
	"energy-monitoring-system/internal/utils"
	"net/http"

)

//"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NzkyNDA4OTQsImlzcyI6ImVuZXJneS1tb25pdG9yaW5nLXN5c3RlbSIsInJvbGUiOiJ1c2VyIiwic3ViIjoiYWNjZXNzIiwidXNlcl9pZCI6ImRiYTYwNDk3LWM5ZDAtNGRkMC1hZWZjLWVkNjQyMjg2Njk3ZiJ9.5UlcnC0lLhs5EUNKB5SrYkqhC4tI68be4T_2KtWwrtI"


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



