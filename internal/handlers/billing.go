package handlers

import (
	"encoding/json"
	"energy-monitoring-system/internal/auth/middleware"
	"energy-monitoring-system/internal/db"
	"energy-monitoring-system/internal/models"
	"energy-monitoring-system/internal/services"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type TopUpRequest struct {
	Amount    float64 `json:"amount"`
	Reference string  `json:"reference"`
}

type SetTariffRequest struct {
	Limit       float64 `json:"limit"`
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
	if req.Amount <= 0 {
		http.Error(w, "Amount must be greater than 0", http.StatusBadRequest)
		return
	}

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		return services.TopUpWallet(tx, userID, req.Amount, req.Reference)
	}); err != nil {
		if errors.Is(err, services.ErrUserInactive) {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Error(w, "Failed to top up wallet: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Wallet topped up successfully"})
}

// AdminTopUpUserHandler lets an admin credit kWh to any user's wallet.
// POST /admin/users/{id}/wallet/topup
func AdminTopUpUserHandler(w http.ResponseWriter, r *http.Request) {
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

	var req TopUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Amount <= 0 {
		http.Error(w, "Amount must be greater than 0", http.StatusBadRequest)
		return
	}

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		return services.TopUpWallet(tx, userID, req.Amount, req.Reference)
	}); err != nil {
		if errors.Is(err, services.ErrUserInactive) {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Wallet not found for this user", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to top up wallet: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "User wallet topped up successfully"})
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
	transaction, err := models.GetTransactionsByWalletID(wallet.ID)
	if err != nil {
		http.Error(w, "Failed to fetch transactions", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(transaction)
}

func GetUserWalletHandler(w http.ResponseWriter, r *http.Request) {
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
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(wallet)
}

func GetUserTransactionsHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Failed to fetch wallet: "+err.Error(), http.StatusInternalServerError)
		return
	}

	transaction, err := models.GetTransactionsByWalletID(wallet.ID)
	if err != nil {
		http.Error(w, "Failed to fetch transactions", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(transaction)
}

func GetTransactionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	transactionIDStr := vars["id"]

	if transactionIDStr == "" {
		http.Error(w, "Missing transaction ID", http.StatusBadRequest)
		return
	}

	transactionID, err := uuid.Parse(transactionIDStr)
	if err != nil {
		http.Error(w, "Invalid transaction ID", http.StatusBadRequest)
		return
	}

	transaction, err := models.GetTransactionByTransactionID(transactionID)
	if err != nil {
		http.Error(w, "Failed to fetch transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(transaction)
}

func AdminSetTariffHandler(w http.ResponseWriter, r *http.Request) {
	var req SetTariffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.PricePerKWh <= 0 {
		http.Error(w, "Price must be positive", http.StatusBadRequest)
		return
	}

	if req.Limit <= 0 {
		http.Error(w, "Limit must be positive", http.StatusBadRequest)
		return
	}
	var tariff = models.TariffTier{
		Limit: req.Limit,
		Rate:  req.PricePerKWh,
	}
	tariff.CreatedAt = time.Now()
	tariff.UpdatedAt = time.Now()
	if err := tariff.Set(); err != nil {
		http.Error(w, "Failed to set tariff: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Tariff set successfully"})
}

func AdminGetTariffsHandler(w http.ResponseWriter, r *http.Request) {
	tariff, err := models.GetTariffTiers()
	if err != nil {
		http.Error(w, "Failed to fetch tariff: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tariff)
}

func CalculatorHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Unit   string  `json:"unit"` // "kwh" or "money"
		Amount float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		http.Error(w, "Amount must be positive", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch req.Unit {
	case "kwh":
		cost, err := models.CalculateCost(req.Amount)
		if err != nil {
			http.Error(w, "Calculation failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"input_unit":   "kwh",
			"input_amount": req.Amount,
			"result_cost":  cost,
		})

	case "money":
		kwh, err := models.CalculatePower(req.Amount)
		if err != nil {
			http.Error(w, "Calculation failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"input_unit":   "money",
			"input_amount": req.Amount,
			"result_kwh":   kwh,
			"message":      "kWh calculated successfully",
		})

	default:
		http.Error(w, "Invalid unit. Use 'kwh' or 'money'", http.StatusBadRequest)
	}
}
