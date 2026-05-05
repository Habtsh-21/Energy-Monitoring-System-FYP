package handlers

import (
	"encoding/json"
	"energy-monitoring-system/internal/models"
	"energy-monitoring-system/internal/services"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LineReadingRequest struct {
	MeterSerialNumber string  `json:"meter_serial_number"`
	PoleCurrentA      float64 `json:"pole_current_a"`
	PoleVoltageV      float64 `json:"pole_voltage_v"`
	MeterCurrentA     float64 `json:"meter_current_a"`
	MeterVoltageV     float64 `json:"meter_voltage_v"`
	RecordedAt        string  `json:"recorded_at"` 
	Note              string  `json:"note"`
}

type LineReadingResponse struct {
	Reading   models.LineReading  `json:"reading"`
	Detection models.BypassResult `json:"detection"`
}

func LineReadingHandler(w http.ResponseWriter, r *http.Request) {
	if key := strings.TrimSpace(os.Getenv("METER_SUBMIT_KEY")); key != "" {
		if r.Header.Get("X-Meter-Key") != key {
			http.Error(w, "Unauthorized meter key", http.StatusUnauthorized)
			return
		}
	}

	var req LineReadingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.MeterSerialNumber) == "" {
		http.Error(w, "meter_serial_number is required", http.StatusBadRequest)
		return
	}
	if req.PoleCurrentA < 0 || req.MeterCurrentA < 0 {
		http.Error(w, "current values cannot be negative", http.StatusBadRequest)
		return
	}
	if req.PoleVoltageV <= 0 || req.MeterVoltageV <= 0 {
		http.Error(w, "voltage values must be positive", http.StatusBadRequest)
		return
	}

	recordedAt := time.Now()
	if ts := strings.TrimSpace(req.RecordedAt); ts != "" {
		parsed, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			http.Error(w, "recorded_at must be RFC3339 format", http.StatusBadRequest)
			return
		}
		recordedAt = parsed
	}

	meterID, err := models.GetMeterIDBySerialNo(req.MeterSerialNumber)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Meter not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to find meter", http.StatusInternalServerError)
		return
	}

	userID, err := models.GetUserIdByMeterId(meterID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "No active user assigned to this meter", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to resolve meter owner", http.StatusInternalServerError)
		return
	}

	lr := models.LineReading{
		MeterID:       meterID,
		UserID:        userID,
		PoleCurrentA:  req.PoleCurrentA,
		PoleVoltageV:  req.PoleVoltageV,
		MeterCurrentA: req.MeterCurrentA,
		MeterVoltageV: req.MeterVoltageV,
		RecordedAt:    recordedAt,
		Note:          strings.TrimSpace(req.Note),
	}
	lr.CreatedAt = time.Now()
	lr.UpdatedAt = time.Now()
	lr.ComputeDerived()

	detection := services.DetectBypass(&lr)

	if err := models.CreateLineReadingWithAnomaly(&lr, detection); err != nil {
		http.Error(w, "Failed to save line reading", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(LineReadingResponse{
		Reading:   lr,
		Detection: detection,
	})
}



func GetLineReadingsByMeterIDHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	meterIDStr := q.Get("meter_id")
	if meterIDStr == "" {
		http.Error(w, "Missing meter_id", http.StatusBadRequest)
		return
	}
	meterID, err := uuid.Parse(meterIDStr)
	if err != nil {
		http.Error(w, "Invalid meter_id", http.StatusBadRequest)
		return
	}

	start, end, limit, offset, ok := parseLineQueryParams(w, r)
	if !ok {
		return
	}

	readings, total, err := models.GetLineReadingsByMeterID(meterID, start, end, limit, offset)
	if err != nil {
		http.Error(w, "Failed to fetch readings", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Readings []models.LineReading `json:"readings"`
		Total    int64                `json:"total"`
	}{Readings: readings, Total: total})
}


func AnalyseMeterHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	meterIDStr := q.Get("meter_id")
	if meterIDStr == "" {
		http.Error(w, "Missing meter_id", http.StatusBadRequest)
		return
	}
	meterID, err := uuid.Parse(meterIDStr)
	if err != nil {
		http.Error(w, "Invalid meter_id", http.StatusBadRequest)
		return
	}

	cfg := services.DefaultAnalyserConfig()
	if windowStr := q.Get("window"); windowStr != "" {
		if w2, err := strconv.Atoi(windowStr); err == nil && w2 > 0 {
			cfg.WindowSize = w2
		}
	}

	analyser := services.NewLineAnalyser(cfg)
	report, err := analyser.AnalyseMeter(meterID)
	if err != nil {
		http.Error(w, "Analysis failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}

func RunNightlyBatchHandler(w http.ResponseWriter, r *http.Request) {
	analyser := services.NewLineAnalyser(services.DefaultAnalyserConfig())
	batch, err := analyser.RunNightlyBatch()
	if err != nil {
		http.Error(w, "Batch analysis failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(batch)
}

func parseLineQueryParams(w http.ResponseWriter, r *http.Request) (
	start, end time.Time, limit, offset int, ok bool,
) {
	q := r.URL.Query()

	startStr := q.Get("start_time")
	endStr := q.Get("end_time")
	limitStr := q.Get("limit")
	offsetStr := q.Get("offset")

	if startStr == "" || endStr == "" {
		http.Error(w, "Missing start_time or end_time", http.StatusBadRequest)
		return
	}
	var err error
	start, err = time.Parse(time.RFC3339, startStr)
	if err != nil {
		http.Error(w, "Invalid start_time (RFC3339 required)", http.StatusBadRequest)
		return
	}
	end, err = time.Parse(time.RFC3339, endStr)
	if err != nil {
		http.Error(w, "Invalid end_time (RFC3339 required)", http.StatusBadRequest)
		return
	}
	if end.Before(start) {
		http.Error(w, "end_time must be after start_time", http.StatusBadRequest)
		return
	}

	limit = 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr != "" {
		if off, err := strconv.Atoi(offsetStr); err == nil && off >= 0 {
			offset = off
		} else {
			http.Error(w, "Invalid offset", http.StatusBadRequest)
			return
		}
	}

	ok = true
	return
}
