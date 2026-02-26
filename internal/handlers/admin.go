package handlers

import (
	"encoding/json"
	"energy-monitoring-system/internal/models"
	"energy-monitoring-system/internal/utils"
	"net/http"
)

func AdminHomeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Welcome to Energy Monitoring System"))
}

func UserRegisterHandler(w http.ResponseWriter, r *http.Request) {

	var user models.User
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
	
	if models.CheckPhoneNumber(user.PhoneNumber) {
		http.Error(w, "Phone number already exists", http.StatusBadRequest)
		return
	}
	user.ID = utils.IdGenerator()
    
	if models.CheckId(user.ID) {
		http.Error(w, "User ID already exists", http.StatusBadRequest)
		return
	}

	user.Password, err = utils.HashPassword(user.Password)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}
	

	if err := user.Create(); err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
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

	userId := r.URL.Query().Get("user_id")
	if userId == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
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

	userId := r.URL.Query().Get("user_id")
	if userId == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
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
    
	if models.CheckMeterSerialNo(meter.SerialNo){
		http.Error(w,"Meter serial number already exists", http.StatusBadRequest)
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

	serialNo := r.URL.Query().Get("serialNo")
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

	serialNo := r.URL.Query().Get("serialNo")
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