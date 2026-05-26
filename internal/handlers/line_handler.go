package handlers

import (
	"encoding/json"
	"energy-monitoring-system/internal/auth/middleware"
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

// LineReadingResponse is the single structured envelope returned for every
// line-reading submission.  The HTTP status is always 200; callers must inspect
// the Status field to determine the outcome.
//
// Status values and their meaning:
//
//	"ok"               – reading accepted, relay state unchanged / healthy
//	"low_balance"      – reading accepted, balance < 1 kWh, relay still ON
//	"no_balance"       – balance exhausted, relay commanded OFF
//	"admin_disabled"   – meter administratively killed by admin, relay commanded OFF
//	"owner_disabled"   – meter deliberately disabled by its owner, relay commanded OFF
//	"inactive_account" – user account suspended by admin, relay commanded OFF
type LineReadingResponse struct {
	Status         string  `json:"status"`
	RelayCommand   string  `json:"relay_command"`
	BalanceKwh     float64 `json:"balance_kwh"`
	Message        string  `json:"message"`
	AdminDisabled  bool    `json:"admin_disabled"`
	OwnerDisabled  bool    `json:"owner_disabled"`
	UserInactive   bool    `json:"user_inactive"`
}

// ok200 writes a 200 JSON response. All structured outcomes use this helper so
// there is exactly one place that sets the status code for business responses.
func ok200(w http.ResponseWriter, resp LineReadingResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
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
			http.Error(w, "No user assigned to this meter", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to resolve meter owner", http.StatusInternalServerError)
		return
	}

	var resp LineReadingResponse

	txErr := db.DB.Transaction(func(tx *gorm.DB) error {
		meterStatus, err := models.GetMeterStatus(meterID)
		if err != nil {
			return fmt.Errorf("failed to fetch meter status: %w", err)
		}

		if req.OwnerDisabled != meterStatus.OwnerDisabled {
			if err := models.SetOwnerDisabled(tx, meterID, req.OwnerDisabled); err != nil {
				return fmt.Errorf("failed to sync owner_disabled status to db: %w", err)
			}
			meterStatus.OwnerDisabled = req.OwnerDisabled
		}

		ownerDisabled := meterStatus.OwnerDisabled || req.OwnerDisabled
		userActive := user.IsActive && !req.UserInactive

		if !req.IsConnected {
			resp, err = handleMeterOff(tx, meterID, user.ID, userActive, req, recordedAt, meterStatus.AdminDisabled, ownerDisabled)
		} else {
			resp, err = handleMeterOn(tx, meterID, user.ID, userActive, req, recordedAt, meterStatus.AdminDisabled, ownerDisabled)
		}
		return err
	})

	if txErr != nil {
		http.Error(w, txErr.Error(), http.StatusInternalServerError)
		return
	}

	// All business outcomes share a single HTTP 200.  The Status field in the
	// body is the authoritative signal for the aggregator.
	ok200(w, resp)
}

// ── handleMeterOff ────────────────────────────────────────────────────────────
// Called when the meter reports is_connected = false.
// We still check for bypass (current flowing while relay is off) and record it,
// then return the appropriate status so the aggregator knows why the relay is off.

func handleMeterOff(tx *gorm.DB, meterID, userID uuid.UUID, userActive bool, req models.LineReadingRequest, recordedAt time.Time, adminDisabled, ownerDisabled bool) (LineReadingResponse, error) {

	detection := services.VerifyReading(&req)
	isBypassed := detection.Verdict == services.VerdictConfirmed || detection.Verdict == services.VerdictSuspect

	if isBypassed {
		lr, err := createLineReading(tx, meterID, userID, req, recordedAt)
		if err != nil {
			return LineReadingResponse{}, fmt.Errorf("failed to save line reading: %w", err)
		}
		if err := createAnomaly(tx, lr.ID, detection, recordedAt); err != nil {
			return LineReadingResponse{}, fmt.Errorf("failed to save anomaly: %w", err)
		}
	}

	wallet, _ := models.GetWalletByUserID(userID)
	bal := 0.0
	if wallet != nil {
		bal = wallet.BalanceKwh
	}

	// Priority: admin kill > owner disabled > balance check > account status > normal off.
	switch {
	case adminDisabled:
		return LineReadingResponse{
			Status:        "admin_disabled",
			RelayCommand:  "OFF",
			BalanceKwh:    bal,
			Message:       "Meter is administratively disabled. Contact your service provider.",
			AdminDisabled: true,
			OwnerDisabled: ownerDisabled,
			UserInactive:  !userActive,
		}, nil

	case ownerDisabled:
		return LineReadingResponse{
			Status:        "owner_disabled",
			RelayCommand:  "OFF",
			BalanceKwh:    bal,
			Message:       "Meter has been disabled by the owner.",
			AdminDisabled: adminDisabled,
			OwnerDisabled: true,
			UserInactive:  !userActive,
		}, nil

	case bal <= 0:
		return LineReadingResponse{
			Status:        "no_balance",
			RelayCommand:  "OFF",
			BalanceKwh:    0,
			Message:       "Balance exhausted. Meter is disconnected.",
			AdminDisabled: adminDisabled,
			OwnerDisabled: ownerDisabled,
			UserInactive:  !userActive,
		}, nil

	case !userActive:
		return LineReadingResponse{
			Status:        "inactive_account",
			RelayCommand:  "OFF",
			BalanceKwh:    bal,
			Message:       "User account is inactive. Relay must stay off.",
			AdminDisabled: adminDisabled,
			OwnerDisabled: ownerDisabled,
			UserInactive:  true,
		}, nil

	default:
		return LineReadingResponse{
			Status:        "ok",
			RelayCommand:  "OFF",
			BalanceKwh:    bal,
			Message:       "Meter is OFF.",
			AdminDisabled: adminDisabled,
			OwnerDisabled: ownerDisabled,
			UserInactive:  !userActive,
		}, nil
	}
}

// ── handleMeterOn ─────────────────────────────────────────────────────────────
// Called when the meter reports is_connected = true.
// Saves the reading, debits the wallet, then evaluates override conditions
// (admin kill, owner disable, inactive account) that must force the relay off regardless.

func handleMeterOn(tx *gorm.DB, meterID, userID uuid.UUID, userActive bool, req models.LineReadingRequest, recordedAt time.Time, adminDisabled, ownerDisabled bool) (LineReadingResponse, error) {

	lr, err := createLineReading(tx, meterID, userID, req, recordedAt)
	if err != nil {
		return LineReadingResponse{}, fmt.Errorf("failed to save line reading: %w", err)
	}

	detection := services.VerifyReading(&req)
	if detection.Verdict == services.VerdictConfirmed || detection.Verdict == services.VerdictSuspect {
		if err := createAnomaly(tx, lr.ID, detection, recordedAt); err != nil {
			return LineReadingResponse{}, fmt.Errorf("failed to save anomaly: %w", err)
		}
	}

	// ── Wallet debit ──────────────────────────────────────────────────────────

	var resp LineReadingResponse
	wallet, err := models.GetWalletByUserID(userID)
	if err != nil {
		return LineReadingResponse{}, fmt.Errorf("failed to fetch wallet: %w", err)
	}
	if req.ConsumedKwh > 0 {
		debitErr := services.DebitWallet(tx, userID, req.ConsumedKwh, lr.ID.String())

		switch {
		case errors.Is(debitErr, services.ErrInsufficientBalance):
			if err := models.TurnOffMeter(tx, meterID); err != nil {
				return LineReadingResponse{}, fmt.Errorf("failed to turn off meter: %w", err)
			}
			return LineReadingResponse{
				Status:        "no_balance",
				RelayCommand:  "OFF",
				BalanceKwh:    wallet.BalanceKwh,
				Message:       "Balance exhausted. Meter has been disconnected.",
				AdminDisabled: adminDisabled,
				OwnerDisabled: ownerDisabled,
				UserInactive:  !userActive,
			}, nil

		case debitErr != nil:
			return LineReadingResponse{}, fmt.Errorf("wallet debit failed: %w", debitErr)

		default:
			const lowBalanceThreshold = 1.0
			if wallet.BalanceKwh < lowBalanceThreshold {
				resp = LineReadingResponse{
					Status:        "low_balance",
					RelayCommand:  "ON",
					BalanceKwh:    wallet.BalanceKwh,
					Message:       "Low balance",
					AdminDisabled: adminDisabled,
					OwnerDisabled: ownerDisabled,
					UserInactive:  !userActive,
				}
			} else {
				resp = LineReadingResponse{
					Status:        "ok",
					RelayCommand:  "ON",
					BalanceKwh:    wallet.BalanceKwh,
					Message:       "Reading recorded.",
					AdminDisabled: adminDisabled,
					OwnerDisabled: ownerDisabled,
					UserInactive:  !userActive,
				}
			}
		}
	} else {
		resp = LineReadingResponse{
			Status:        "ok",
			RelayCommand:  "ON",
			BalanceKwh:    wallet.BalanceKwh,
			Message:       "Reading recorded. No energy consumed.",
			AdminDisabled: adminDisabled,
			OwnerDisabled: ownerDisabled,
			UserInactive:  !userActive,
		}
	}

	// ── Override checks (evaluated after debit so usage is always recorded) ──
	// Priority: admin kill > owner disabled > inactive account. Both force the relay OFF
	// regardless of what the debit path decided above.

	switch {
	case adminDisabled:
		return LineReadingResponse{
			Status:        "admin_disabled",
			RelayCommand:  "OFF",
			BalanceKwh:    wallet.BalanceKwh,
			Message:       "Meter is administratively disabled. Contact your service provider.",
			AdminDisabled: true,
			OwnerDisabled: ownerDisabled,
			UserInactive:  !userActive,
		}, nil

	case ownerDisabled:
		return LineReadingResponse{
			Status:        "owner_disabled",
			RelayCommand:  "OFF",
			BalanceKwh:    wallet.BalanceKwh,
			Message:       "Meter has been disabled by the owner.",
			AdminDisabled: adminDisabled,
			OwnerDisabled: true,
			UserInactive:  !userActive,
		}, nil

	case !userActive:
		return LineReadingResponse{
			Status:        "inactive_account",
			RelayCommand:  "OFF",
			BalanceKwh:    wallet.BalanceKwh,
			Message:       "Account inactive: usage was debited from the wallet; relay remains off.",
			AdminDisabled: adminDisabled,
			OwnerDisabled: ownerDisabled,
			UserInactive:  true,
		}, nil
	}

	return resp, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

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



func GetUserReadingHandler(w http.ResponseWriter, r *http.Request){
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	reading, err := models.GetUserReading(userID);
	if(err != nil) {
		http.Error(w, "Failed to fetch reading", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(reading)

}