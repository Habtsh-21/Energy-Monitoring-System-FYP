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
	"gorm.io/gorm"
)

type MeterReadingRequest struct {
	MeterSerialNumber string      `json:"meter_serial_number"`
	ReadingKWh        FlexFloat64 `json:"reading_kwh"`
	ReadAt            string      `json:"read_at"`
	Note              string      `json:"note"`
}

var defaultLimit int = 10
var defaultOffset int = 0

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
		http.Error(w, "Invalid request body"+err.Error(), http.StatusBadRequest)
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

	userId, err := models.GetUserIdByMeterId(meterId)
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

func GetAllMeterReadingHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	startTimeStr := query.Get("start_time")
	endTimeStr := query.Get("end_time")
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")

	// Validate limit
	if limitStr == "" {
		http.Error(w, "Missing limit", http.StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		http.Error(w, "Invalid limit", http.StatusBadRequest)
		return
	}

	// Enforce safe limit
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	// Optional offset
	offset := 0
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			http.Error(w, "Invalid offset", http.StatusBadRequest)
			return
		}
	}

	// Validate time
	if startTimeStr == "" || endTimeStr == "" {
		http.Error(w, "Missing start_time or end_time", http.StatusBadRequest)
		return
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		http.Error(w, "Invalid start_time format (use RFC3339)", http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		http.Error(w, "Invalid end_time format (use RFC3339)", http.StatusBadRequest)
		return
	}

	if endTime.Before(startTime) {
		http.Error(w, "end_time must be after start_time", http.StatusBadRequest)
		return
	}

	readings, total, err := models.GetAllMeterReadings(startTime, endTime, limit, offset)
	if err != nil {
		http.Error(w, "Failed to get readings", http.StatusInternalServerError)
		return
	}

	response := struct {
		Readings []models.MeterReading `json:"readings"`
		Total    int64                 `json:"total"`
	}{
		Readings: readings,
		Total:    total,
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}



func GetMeterReadingByMeterIDHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	meterIDStr := query.Get("meter_id")
	startTimeStr := query.Get("start_time")
	endTimeStr := query.Get("end_time")
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")

	if meterIDStr == "" {
		http.Error(w, "Missing meter_id", http.StatusBadRequest)
		return
	}

	meterID, err := uuid.Parse(meterIDStr)
	if err != nil {
		http.Error(w, "Invalid meter_id", http.StatusBadRequest)
		return
	}

	if limitStr == "" {
		http.Error(w, "Missing limit", http.StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		http.Error(w, "Invalid limit", http.StatusBadRequest)
		return
	}

	if limit <= 0 || limit > 100 {
		limit = 10
	}

	offset := 0
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			http.Error(w, "Invalid offset", http.StatusBadRequest)
			return
		}
	}

	if startTimeStr == "" || endTimeStr == "" {
		http.Error(w, "Missing start_time or end_time", http.StatusBadRequest)
		return
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		http.Error(w, "Invalid start_time format", http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		http.Error(w, "Invalid end_time format", http.StatusBadRequest)
		return
	}

	if endTime.Before(startTime) {
		http.Error(w, "end_time must be after start_time", http.StatusBadRequest)
		return
	}

	readings, total, err := models.GetMeterReadingByMeterID(meterID, startTime, endTime, limit, offset)
	if err != nil {
		http.Error(w, "Failed to get readings", http.StatusInternalServerError)
		return
	}

	response := struct {
		Readings []models.MeterReading `json:"readings"`
		Total    int64                 `json:"total"`
	}{
		Readings: readings,
		Total:    total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}



func GetMeterReadingByUserIDHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	userIDStr := query.Get("user_id")
	startTimeStr := query.Get("start_time")
	endTimeStr := query.Get("end_time")
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")

	// Validate user_id
	if userIDStr == "" {
		http.Error(w, "Missing user_id", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Validate limit
	if limitStr == "" {
		http.Error(w, "Missing limit", http.StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		http.Error(w, "Invalid limit", http.StatusBadRequest)
		return
	}

	// Enforce safe limit
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	// Optional offset
	offset := 0
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			http.Error(w, "Invalid offset", http.StatusBadRequest)
			return
		}
	}

	// Validate time
	if startTimeStr == "" || endTimeStr == "" {
		http.Error(w, "Missing start_time or end_time", http.StatusBadRequest)
		return
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		http.Error(w, "Invalid start_time format", http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		http.Error(w, "Invalid end_time format", http.StatusBadRequest)
		return
	}

	if endTime.Before(startTime) {
		http.Error(w, "end_time must be after start_time", http.StatusBadRequest)
		return
	}

	// DB call
	readings, total, err := models.GetMeterReadingByUserID(userID, startTime, endTime, limit, offset)
	if err != nil {
		http.Error(w, "Failed to get readings", http.StatusInternalServerError)
		return
	}

	response := struct {
		Readings []models.MeterReading `json:"readings"`
		Total    int64                 `json:"total"`
	}{
		Readings: readings,
		Total:    total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}







func createAnomaly(reading models.MeterReading, anomalyType, reason string) error {

	anomaly := models.Anomaly{
		ReadingID:  reading.ID,
		Type:       anomalyType,
		Reason:     reason,
		Status:     models.AnomalyStatusOpen,
		DetectedAt: time.Now(),
	}
	anomaly.CreatedAt = time.Now()
	anomaly.UpdatedAt = time.Now()

	return anomaly.Create()
}
