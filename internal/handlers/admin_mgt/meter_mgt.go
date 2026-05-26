package admin_mgt

import (
	"encoding/json"
	"energy-monitoring-system/internal/models"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func MeterRegisterHandler(w http.ResponseWriter, r *http.Request) {

	var meter models.Meter
	var req models.MeterRegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	fmt.Println("--------------------------------------------------------------", req)
	meter.MeterSerialNumber = req.MeterSerialNumber
	meter.MeterType = req.MeterType
	meter.Manufacturer = req.Manufacturer
	meter.Model = req.Model
	meter.FirmwareVersion = req.FirmwareVersion

	if meter.MeterSerialNumber == "" || meter.MeterType == "" {
		http.Error(w, "Incomplete meter information", http.StatusBadRequest)
		return
	}
	meter.CreatedAt = time.Now()
	meter.UpdatedAt = time.Now()

	if models.CheckSerialNo(meter.MeterSerialNumber) {
		http.Error(w, "Meter serial number already exists", http.StatusConflict)
		return
	}

	if err := meter.Create(); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key value violates unique constraint") {
			http.Error(w, "Meter serial number already exists", http.StatusConflict)
			return
		}
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

	vars := mux.Vars(r)
	meterID := vars["id"]
	if meterID == "" {
		http.Error(w, "Meter ID is required", http.StatusBadRequest)
		return
	}

	meter, err := models.GetMeterByID(uuid.Must(uuid.Parse(meterID)))
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

	if strings.EqualFold(meter.RelayStatus, "ON") && meter.ID != uuid.Nil {
		u, err := models.GetUserByMeterID(meter.ID)
		if err == nil && !u.IsActive {
			http.Error(w, "Cannot set relay ON: assigned user account is inactive", http.StatusForbidden)
			return
		}
	}

	if err := meter.Update(); err != nil {
		http.Error(w, "Failed to update meter", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func DeleteMeterHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	meterID := vars["id"]
	if meterID == "" {
		http.Error(w, "Meter ID is required", http.StatusBadRequest)
		return
	}

	isAssigned, err := models.IsMeterAssigned(uuid.Must(uuid.Parse(meterID)))
	if err != nil {
		http.Error(w, "Failed to get meter", http.StatusInternalServerError)
		return
	}

	if isAssigned {
		http.Error(w, "Meter is assigned to a user", http.StatusBadRequest)
		return
	}

	meter, err := models.GetMeterByID(uuid.Must(uuid.Parse(meterID)))
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

func AdminControlMeterHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	meterIDStr := vars["id"]
	if meterIDStr == "" {
		http.Error(w, "Meter ID is required", http.StatusBadRequest)
		return
	}
	meterID, err := uuid.Parse(meterIDStr)
	if err != nil {
		http.Error(w, "Invalid meter ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Disabled bool `json:"disabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !req.Disabled {
		u, err := models.GetUserByMeterID(meterID)
		if err == nil && !u.IsActive {
			http.Error(w, "Cannot enable meter: assigned user account is inactive", http.StatusForbidden)
			return
		}
	}

	if err := models.SetMeterStatus(nil, meterID, req.Disabled); err != nil {
		if errors.Is(err, models.ErrMeterDisabledByOwner) {
			http.Error(w, "Cannot enable meter: disabled by owner", http.StatusForbidden)
			return
		}
		http.Error(w, "Failed to update meter control: "+err.Error(), http.StatusInternalServerError)
		return
	}

	msg := "Meter enabled by admin"
	if req.Disabled {
		msg = "Meter disabled by admin — relay forced OFF"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": msg})
}
