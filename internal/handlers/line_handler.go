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
	Status       string  `json:"status"`
	RelayCommand string  `json:"relay_command"`
	BalanceKwh   float64 `json:"balance_kwh"`
	Message      string  `json:"message"`
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
	if req.ConsumedKwh < 0 {
		http.Error(w, "consumed_kwh cannot be negative", http.StatusBadRequest)
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

	user, err := models.GetUserByMeterID(meterID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "No user assigned to this meter", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to resolve meter owner", http.StatusInternalServerError)
		return
	}
	userID := user.ID
	userActive := user.IsActive
	var resp LineReadingResponse
	var httpStatus int

	txErr := db.DB.Transaction(func(tx *gorm.DB) error {
		meterStatus, err := models.GetMeterStatus(meterID)
		if err != nil {
			return fmt.Errorf("failed to fetch meter status: %w", err)
		}
		adminDisabled := meterStatus.AdminDisabled

		if !req.IsConnected {
			resp, httpStatus, err = handleMeterOff(tx, meterID, userID, userActive, req, recordedAt, adminDisabled)
		} else {
			resp, httpStatus, err = handleMeterOn(tx, meterID, userID, userActive, req, recordedAt, adminDisabled)
		}
		return err
	})

	if txErr != nil {
		http.Error(w, txErr.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(resp)
}

func handleMeterOff(tx *gorm.DB, meterID, userID uuid.UUID, userActive bool, req models.LineReadingRequest, recordedAt time.Time, adminDisabled bool) (LineReadingResponse, int, error) {
	var resp LineReadingResponse
	httpStatus := http.StatusOK

	detection := services.VerifyReading(&req)
	isBypassed := detection.Verdict == services.VerdictConfirmed || detection.Verdict == services.VerdictSuspect

	if isBypassed {
		lr, err := createLineReading(tx, meterID, userID, req, recordedAt)
		if err != nil {
			return resp, httpStatus, fmt.Errorf("failed to save line reading: %w", err)
		}

		if err := createAnomaly(tx, lr.ID, detection, recordedAt); err != nil {
			return resp, httpStatus, fmt.Errorf("failed to save anomaly: %w", err)
		}
	}

	wallet, _ := models.GetWalletByUserID(userID)
	bal := 0.0
	if wallet != nil {
		bal = wallet.BalanceKwh
	}

	statusMsg := "ok"
	msg := "Meter is OFF."
	if adminDisabled {
		statusMsg = "admin_disabled"
		msg = "Meter is administratively disabled. Contact your service provider."
		httpStatus = http.StatusForbidden
	} else if bal <= 0 {
		statusMsg = "no_balance"
		msg = "Balance exhausted. Meter is disconnected."
		httpStatus = http.StatusPaymentRequired
	} else if !userActive {
		statusMsg = "inactive_account"
		msg = "User account is inactive. Relay must stay off."
		httpStatus = http.StatusForbidden
	}

	resp = LineReadingResponse{
		Status:       statusMsg,
		RelayCommand: "OFF",
		BalanceKwh:   bal,
		Message:      msg,
	}

	return resp, httpStatus, nil
}

func handleMeterOn(tx *gorm.DB, meterID, userID uuid.UUID, userActive bool, req models.LineReadingRequest, recordedAt time.Time, adminDisabled bool) (LineReadingResponse, int, error) {
	var resp LineReadingResponse
	httpStatus := http.StatusOK

	lr, err := createLineReading(tx, meterID, userID, req, recordedAt)
	if err != nil {
		return resp, httpStatus, fmt.Errorf("failed to save line reading: %w", err)
	}
	detection := services.VerifyReading(&req)
	if detection.Verdict == services.VerdictConfirmed || detection.Verdict == services.VerdictSuspect {
		if err := createAnomaly(tx, lr.ID, detection, recordedAt); err != nil {
			return resp, httpStatus, fmt.Errorf("failed to save anomaly: %w", err)
		}
	}

	if req.ConsumedKwh > 0 {
		debitErr := services.DebitWallet(tx, userID, req.ConsumedKwh, lr.ID.String())

		if errors.Is(debitErr, services.ErrInsufficientBalance) {
			if err := models.TurnOffMeter(tx, meterID); err != nil {
				return resp, httpStatus, fmt.Errorf("failed to turn off meter: %w", err)
			}
			resp = LineReadingResponse{
				Status:       "no_balance",
				RelayCommand: "OFF",
				BalanceKwh:   0,
				Message:      "Balance exhausted. Meter has been disconnected.",
			}
			httpStatus = http.StatusPaymentRequired
		} else if debitErr != nil {
			return resp, httpStatus, fmt.Errorf("wallet debit failed: %w", debitErr)
		} else {
			wallet, err := models.GetWalletByUserID(userID)
			if err != nil {
				return resp, httpStatus, fmt.Errorf("failed to fetch wallet: %w", err)
			}

			const lowBalanceThreshold = 1.0
			if wallet.BalanceKwh < lowBalanceThreshold {
				resp = LineReadingResponse{
					Status:       "low_balance",
					RelayCommand: "ON",
					BalanceKwh:   wallet.BalanceKwh,
					Message:      fmt.Sprintf("Low balance: %.4f kWh remaining. Please top up soon.", wallet.BalanceKwh),
				}
			} else {
				resp = LineReadingResponse{
					Status:       "ok",
					RelayCommand: "ON",
					BalanceKwh:   wallet.BalanceKwh,
					Message:      "Reading recorded.",
				}
			}
			httpStatus = http.StatusCreated
		}
	} else {

		resp = LineReadingResponse{
			Status:       "ok",
			RelayCommand: "ON",
			Message:      "Reading recorded. No energy consumed.",
		}
		httpStatus = http.StatusCreated
	}

	if adminDisabled {
		if err := models.TurnOffMeter(tx, meterID); err != nil {
			return resp, httpStatus, fmt.Errorf("failed to enforce admin relay OFF: %w", err)
		}
		resp = LineReadingResponse{
			Status:       "admin_disabled",
			RelayCommand: "OFF",
			BalanceKwh:   resp.BalanceKwh,
			Message:      "Meter is administratively disabled. Contact your service provider.",
		}
		httpStatus = http.StatusForbidden
	} else if !userActive {
		// Usage was still debited above; inactive accounts must not energize the relay.
		if err := models.TurnOffMeter(tx, meterID); err != nil {
			return resp, httpStatus, fmt.Errorf("failed to enforce inactive relay OFF: %w", err)
		}
		bal := resp.BalanceKwh
		var w models.Wallet
		if err := tx.Where("user_id = ?", userID).First(&w).Error; err == nil {
			bal = w.BalanceKwh
		}
		resp = LineReadingResponse{
			Status:       "inactive_account",
			RelayCommand: "OFF",
			BalanceKwh:   bal,
			Message:      "Account inactive: usage was debited from the wallet; relay remains off.",
		}
	}

	return resp, httpStatus, nil
}

func createLineReading(tx *gorm.DB, meterID, userID uuid.UUID, req models.LineReadingRequest, recordedAt time.Time) (*models.LineReading, error) {
	lr := models.LineReading{
		MeterID:              meterID,
		UserID:               userID,
		PoleCurrentA:         req.PoleCurrentA,
		PoleVoltageV:         req.PoleVoltageV,
		MeterCurrentA:        req.MeterCurrentA,
		MeterVoltageV:        req.MeterVoltageV,
		PoleApparentPowerVA:  req.PolePowerVA,
		MeterApparentPowerVA: req.MeterPowerVA,
		DeltaCurrentA:        req.PoleCurrentA - req.MeterCurrentA,
		DeltaVoltageV:        req.PoleVoltageV - req.MeterVoltageV,
		PowerLossPct:         req.PowerLossPct,
		RecordedAt:           recordedAt,
		Note:                 strings.TrimSpace(req.Note),
	}
	lr.CreatedAt = time.Now()
	lr.UpdatedAt = time.Now()
	if err := lr.Create(tx); err != nil {
		return nil, err
	}
	return &lr, nil
}

func createAnomaly(tx *gorm.DB, readingID uuid.UUID, detection services.VerificationResult, recordedAt time.Time) error {
	anomaly := models.Anomaly{
		ReadingID:      readingID,
		Verdict:        string(detection.Verdict),
		MeterClaim:     detection.MeterClaim,
		OurDetection:   detection.OurDetection,
		Conflict:       detection.Conflict,
		ConflictReason: detection.ConflictReason,
		Signals:        detection.Signals,
		Reason:         detection.Reason,
		DetectedAt:     recordedAt,
	}
	return anomaly.Create(tx)
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
