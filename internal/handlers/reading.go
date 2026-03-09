package handlers

import (
	"encoding/json"
	"energy-monitoring-system/internal/models"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type MeterReadingRequest struct {
	MeterSerialNumber string  `json:"meter_serial_number"`
	ReadingKWh        FlexFloat64 `json:"reading_kwh"`
	ReadAt            string  `json:"read_at"`
	Note              string  `json:"note"`
}

type FlexFloat64 float64

func (f *FlexFloat64) UnmarshalJSON(data []byte) error {
	var fl float64
	if err := json.Unmarshal(data, &fl); err == nil {
		*f = FlexFloat64(fl)
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("reading_kwh must be a number or numeric string")
	}

	parsed, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return fmt.Errorf("reading_kwh is not a valid number: %w", err)
	}

	*f = FlexFloat64(parsed)
	return nil
}


func MeterReadingHandler(w http.ResponseWriter, r *http.Request) {
	expectedKey := strings.TrimSpace(os.Getenv("METER_SUBMIT_KEY"))
	
	if expectedKey != "" {
		if r.Header.Get("X-Meter-Key") != expectedKey {
			http.Error(w, "Unauthorized meter key", http.StatusUnauthorized)
			return
		}
	}

	var req MeterReadingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body" + err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.MeterSerialNumber) == "" {
		http.Error(w, "meter_serial_number is required", http.StatusBadRequest)
		return
	}
	if req.ReadingKWh < 0 {
		http.Error(w, "reading_kwh cannot be negative", http.StatusBadRequest)
		return
	}

	readAt := time.Now()
	if strings.TrimSpace(req.ReadAt) != "" {
		parsed, err := time.Parse(time.RFC3339, req.ReadAt)
		if err != nil {
			http.Error(w, "read_at must be RFC3339 format", http.StatusBadRequest)
			return
		}
		readAt = parsed
	}

	meterId, err := models.GetMeterIDBySerialNo(req.MeterSerialNumber)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Meter not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to find meter", http.StatusInternalServerError)
		return
	}

	userId,err := models.GetUserIdByMeterId(meterId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "No active user assigned to this meter", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to resolve meter owner", http.StatusInternalServerError)
		return
	}

	reading := models.MeterReading{
		MeterID:    meterId,
		UserID:     userId,
		ReadingKWh: float64(req.ReadingKWh),
		ReadAt:     readAt,
		Note:       strings.TrimSpace(req.Note),
	}
	reading.CreatedAt = time.Now()
	reading.UpdatedAt = time.Now()

	if err := reading.Create(nil); err != nil {
		http.Error(w, "Failed to save meter reading", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(reading)
}

 func GetMeterReadingHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	meterId, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid meter ID", http.StatusBadRequest)
		return
	}
	
	readings, err := models.GetMeterReadingByMeterID(meterId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "No readings found for this meter", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get readings", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(readings)
}


func GetUserReadingHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	userId, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	
	readings, err := models.GetMeterReadingByUserID(userId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "No readings found for this user", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get readings", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(readings)
}