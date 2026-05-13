package models

import (
	"energy-monitoring-system/internal/db"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	ReportStatusOpen       = "open"
	ReportStatusInProgress = "in_progress"
	ReportStatusResolved   = "resolved"
	ReportStatusRejected   = "rejected"
)

type Report struct {
	BaseModel
	UserID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	Title       string     `gorm:"size:150;not null" json:"title"`
	Category    string     `gorm:"size:50;not null;index" json:"category"`
	Description string     `gorm:"type:text;not null" json:"description"`
	Status      string     `gorm:"size:20;not null;default:open;index" json:"status"`
	Priority    string     `gorm:"size:20;not null;default:normal" json:"priority"`
	AdminNote   string     `gorm:"type:text" json:"admin_note,omitempty"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	User        *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type CreateReportRequest struct {
	Title       string `json:"title"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
}

type UpdateReportStatusRequest struct {
	Status    string `json:"status"`
	AdminNote string `json:"admin_note"`
}

func NormalizeReportStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case ReportStatusInProgress:
		return ReportStatusInProgress
	case ReportStatusResolved:
		return ReportStatusResolved
	case ReportStatusRejected:
		return ReportStatusRejected
	default:
		return ReportStatusOpen
	}
}

func (report *Report) Create() error {
	return db.DB.Create(report).Error
}
 
func GetUserReports(userID uuid.UUID) ([]Report, error) {
	reports := make([]Report, 0)
	err := db.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&reports).Error
	if err != nil {
		return nil, err
	}
	return reports, nil
}

func GetReportByID(id uuid.UUID) (*Report, error) {
	var report Report
	err := db.DB.Preload("User").Where("id = ?", id).First(&report).Error
	if err != nil {
		return nil, err 
	}
	return &report, nil
}

func GetUserReportByID(userID, reportID uuid.UUID) (*Report, error) {
	var report Report
	err := db.DB.Where("id = ? AND user_id = ?", reportID, userID).First(&report).Error
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func GetAllReports() ([]Report, error) {
	reports := make([]Report, 0)
	err := db.DB.Order("created_at DESC").Find(&reports).Error
	if err != nil {
		return nil, err
	}
	return reports, nil
}

func UpdateReportStatus(reportID uuid.UUID, status, adminNote string) error {
	updates := map[string]any{
		"status":     NormalizeReportStatus(status),
		"admin_note": strings.TrimSpace(adminNote),
	}

	normalized := updates["status"].(string)
	if normalized == ReportStatusResolved {
		now := time.Now()
		updates["resolved_at"] = &now
	} else {
		updates["resolved_at"] = nil
	}

	return db.DB.Model(&Report{}).Where("id = ?", reportID).Updates(updates).Error
}
