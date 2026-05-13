package handlers

import (
	"encoding/json"
	"energy-monitoring-system/internal/db"
	"energy-monitoring-system/internal/models"
	"energy-monitoring-system/internal/services"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

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
	var req models.LineReadingRequest
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

	err = db.DB.Transaction(func(tx *gorm.DB) error {

		lr := models.LineReading{
			MeterID:       meterID,
			UserID:        userID,
			PoleCurrentA:  req.PoleCurrentA,
			PoleVoltageV:  req.PoleVoltageV,
			MeterCurrentA: req.MeterCurrentA,
			MeterVoltageV: req.MeterVoltageV, 
			PoleApparentPowerVA: req.PolePowerVA,
			MeterApparentPowerVA: req.MeterPowerVA, 
			DeltaCurrentA: req.PoleCurrentA - req.MeterCurrentA, 
			DeltaVoltageV: req.PoleVoltageV - req.MeterVoltageV,
			PowerLossPct: req.PowerLossPct,
			RecordedAt:    recordedAt,
			Note:          strings.TrimSpace(req.Note),
		}
		lr.CreatedAt = time.Now()
		lr.UpdatedAt = time.Now()
		
 
		if err := lr.Create(tx); err != nil {
			return fmt.Errorf("failed to save line reading: %w", err)
		}
		

		// update wallet balance
		if err := services.UpdateWallet(userID, req.ConsumedKwh, models.TxTypeUsageDebit, ""); err != nil {
			return fmt.Errorf("failed to update wallet: %w", err)
		}
		
		

		detection := services.VerifyReading(&req)

		// if anomaly
		if detection.Verdict == services.VerdictConfirmed || detection.Verdict == services.VerdictSuspect {
			anomaly := models.Anomaly{
				ReadingID:      lr.ID,
				Verdict:        string(detection.Verdict),
				MeterClaim:     detection.MeterClaim,
				OurDetection:   detection.OurDetection,
				Conflict:       detection.Conflict,
				ConflictReason: detection.ConflictReason,
				Signals:        detection.Signals,
				Reason:         detection.Reason,
				DetectedAt:     lr.RecordedAt,
			}
			if err := anomaly.Create(tx); err != nil {
				return fmt.Errorf("failed to save anomaly: %w", err)
			}
		}
		return nil
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	

}

func GetMetersReadings(w http.ResponseWriter, r *http.Request) {

	readings, err := models.GetLineReadings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Readings []models.LineReadingsResponse `json:"readings"`
	}{Readings: readings})
}

func GetReadingRecord(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	readingIDStr := vars["id"]

	if readingIDStr == "" {
		http.Error(w, "Missing reading ID", http.StatusBadRequest)
		return
	}

	readingID, err := uuid.Parse(readingIDStr)
	if err != nil {
		http.Error(w, "Invalid reading ID", http.StatusBadRequest)
		return
	}

	reading, err := models.GetReadingRecord(readingID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Reading not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch reading", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(reading)
}
