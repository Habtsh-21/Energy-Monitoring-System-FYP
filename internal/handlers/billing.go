package handlers

import (
	"encoding/json"
	"energy-monitoring-system/internal/auth/middleware"
	//"energy-monitoring-system/internal/db"
	"energy-monitoring-system/internal/models"
	"energy-monitoring-system/internal/services"
	"net/http"
//	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type TopUpRequest struct {
	Amount    float64 `json:"amount"`
	Reference string  `json:"reference"`
}

type SetTariffRequest struct {
	Limit float64 `json:"limit"`
	PricePerKWh float64 `json:"price_per_kwh"`
}

type CalculateKwhRequest struct {
	Cost float64 `json:"cost"`
}
type CalculateCostRequest struct {
	Kwh float64 `json:"kwh"`
}


func WalletTopUpHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req TopUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := services.UpdateWallet(userID, req.Amount, models.TxTypeTopUp,req.Reference); err != nil {
		http.Error(w, "Failed to top up wallet: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Wallet topped up successfully"))
}

func GetWalletBalanceHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	wallet, err := models.GetWalletByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to fetch wallet", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wallet)
}



func GetAllTransactionHandler(w http.ResponseWriter, r *http.Request) {
	transactions, err := models.GetAllTransaction()
	if err != nil {
		http.Error(w, "Failed to fetch transactions", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(transactions)
}




func GetWalletTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDStr := vars["id"]

	if userIDStr == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	} 
 
	wallet, err := models.GetWalletByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to fetch wallet", http.StatusInternalServerError)
		return
	} 
    transaction,err := models.GetTransactionsByWalletID(wallet.ID)
	if err != nil {
		http.Error(w, "Failed to fetch transactions", http.StatusInternalServerError)
		return
	} 
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(transaction)
}

 
// func AdminSetTariffHandler(w http.ResponseWriter, r *http.Request) {
// 	var req SetTariffRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		http.Error(w, "Invalid request body", http.StatusBadRequest)
// 		return
// 	}

// 	if req.PricePerKWh <= 0 {
// 		http.Error(w, "Price must be positive", http.StatusBadRequest)
// 		return
// 	}
    
// 	if req.Limit <= 0 {
// 		http.Error(w, "Limit must be positive", http.StatusBadRequest)
// 		return
// 	}	
// 	var tariff = models.TariffTier{
// 	   Limit: req.Limit,
// 	   Rate: req.PricePerKWh,
// 	}
// 	tariff.CreatedAt = time.Now()
// 	tariff.UpdatedAt = time.Now()
// 	if err := tariff.Set(); err != nil {
// 		http.Error(w, "Failed to set tariff: "+err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	w.WriteHeader(http.StatusOK)
// 	w.Write([]byte("Tariff setted successfully"))
// }


// }
// func AdminGetTariffsHandler(w http.ResponseWriter, r *http.Request){

// 	tariff, err := models.GetTariffTiers()
// 	if err != nil {
// 		http.Error(w, "Failed to fetch tariff: "+err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	w.Header().Set("Content-Type", "application/json")
// 	_ = json.NewEncoder(w).Encode(tariff)
// }


// func CalculateCostHandler(w http.ResponseWriter, r *http.Request) {
// 	var req CalculateCostRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		http.Error(w, "Invalid request body", http.StatusBadRequest)
// 		return
// 	}
// 	if req.Kwh <= 0 {
// 		http.Error(w, "Kwh must be positive", http.StatusBadRequest)
// 		return
// 	}	 
// 	cost, err := models.CalculateCost(req.Kwh) 
// 	if err != nil { 
// 		http.Error(w, "Failed to calculate cost: "+err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	w.Header().Set("Content-Type", "application/json")
// 	_ = json.NewEncoder(w).Encode(cost)
// 	w.WriteHeader(http.StatusOK)
// 	w.Write([]byte("Cost calculated successfully"))
// }



// func CalculateKwhHandler(w http.ResponseWriter, r *http.Request) {
// 	var req CalculateKwhRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		http.Error(w, "Invalid request body", http.StatusBadRequest)
// 		return
// 	}

// 	if req.Cost <= 0 {
// 		http.Error(w, "Cost must be positive", http.StatusBadRequest)
// 		return
// 	}
// 	kwh, err := models.CalculatePower(req.Cost)
// 	if err != nil {
// 		http.Error(w, "Failed to check kwh: "+err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	w.Header().Set("Content-Type", "application/json")
// 	_ = json.NewEncoder(w).Encode(kwh)
// 	w.WriteHeader(http.StatusOK)
// 	w.Write([]byte("Kwh checked successfully"))
// }