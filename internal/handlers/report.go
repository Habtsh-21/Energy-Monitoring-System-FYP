package handlers

import (
	"encoding/json"
	"energy-monitoring-system/internal/auth/middleware"
	"energy-monitoring-system/internal/models"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func CreateReportHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	role, _ := r.Context().Value(middleware.RoleKey).(string)
	if role != "user" {
		http.Error(w, "User token required", http.StatusUnauthorized)
		return
	}

	var req models.CreateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Category = strings.TrimSpace(strings.ToLower(req.Category))
	req.Description = strings.TrimSpace(req.Description)
	req.Priority = strings.TrimSpace(strings.ToLower(req.Priority))

	if req.Title == "" || req.Category == "" || req.Description == "" {
		http.Error(w, "title, category and description are required", http.StatusBadRequest)
		return
	}
	if req.Priority == "" {
		req.Priority = "normal"
	}

	report := models.Report{
		UserID:      userID,
		Title:       req.Title,
		Category:    req.Category,
		Description: req.Description,
		Priority:    req.Priority,
		Status:      models.ReportStatusOpen,
	}
	report.CreatedAt = time.Now()
	report.UpdatedAt = time.Now()

	if err := report.Create(); err != nil {
		http.Error(w, "Failed to create report "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(report)
}

func GetAllReportsHandler(w http.ResponseWriter, r *http.Request) {
	reports, err := models.GetAllReports()
	if err != nil {
		http.Error(w, "Failed to fetch reports", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Reports []models.Report `json:"reports"`
	}{Reports: reports})
}

func GetReportByIDHandler(w http.ResponseWriter, r *http.Request) {
	reportID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid report ID", http.StatusBadRequest)
		return
	}

	report, err := models.GetReportByID(reportID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Report not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch report", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}

func UpdateReportStatusHandler(w http.ResponseWriter, r *http.Request) {
	reportID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid report ID", http.StatusBadRequest)
		return
	}

	var req models.UpdateReportStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Status = strings.TrimSpace(strings.ToLower(req.Status))
	if req.Status == "" {
		http.Error(w, "status is required", http.StatusBadRequest)
		return
	}

	if err := models.UpdateReportStatus(reportID, req.Status, req.AdminNote); err != nil {
		http.Error(w, "Failed to update report status", http.StatusInternalServerError)
		return
	}

	report, err := models.GetReportByID(reportID)
	if err != nil {
		http.Error(w, "Report status updated but failed to fetch report", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}
