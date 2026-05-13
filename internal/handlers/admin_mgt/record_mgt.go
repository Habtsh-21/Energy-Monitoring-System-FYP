
package admin_mgt

import (
	"encoding/json"
	"energy-monitoring-system/internal/models"
	"net/http"
)

func GetAllRecordHandler(w http.ResponseWriter, r *http.Request) {
	records, err := models.GetAllRecord()
	if err != nil {
		http.Error(w, "Failed to get records", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}



