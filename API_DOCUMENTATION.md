


	admin.HandleFunc("/reports", handlers.GetAllReportsHandler).Methods("GET")
	admin.HandleFunc("/reports/{id}", handlers.GetReportByIDHandler).Methods("GET")
	admin.HandleFunc("/reports/{id}/status", handlers.UpdateReportStatusHandler).Methods("PATCH")

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


func GetReportByID(id uuid.UUID) (*Report, error) {
	var report Report
	err := db.DB.Preload("User").Where("id = ?", id).First(&report).Error
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