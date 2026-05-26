package models

import (
	"energy-monitoring-system/internal/db"
	"fmt"
	"sort"

	//"sort"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Wallet struct {
	BaseModel
	UserID  uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	BalanceKwh float64   `gorm:"not null;default:0" json:"balance"`
	Status  string    `gorm:"size:20;default:'active'" json:"status"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type TransactionType string

const (
	TxTypeTopUp      TransactionType = "topup"
	TxTypeUsageDebit TransactionType = "usage_debit"
)

type Transaction struct {
	BaseModel
	WalletID      uuid.UUID       `gorm:"type:uuid;not null;index" json:"wallet_id"`
	Amount        float64         `gorm:"not null" json:"amount"`
	Type          TransactionType `gorm:"size:30;not null;index" json:"type"`
	Reference     string          `gorm:"size:100" json:"reference"`
	Note          string          `gorm:"size:255" json:"note"`
	KWh           float64         `gorm:"column:k_wh;not null;default:0" json:"kwh"`
	BalanceBefore float64         `gorm:"not null;default:0" json:"balance_before"`
	BalanceAfter  float64         `gorm:"not null;default:0" json:"balance_after"`

	Wallet *Wallet `gorm:"foreignKey:WalletID" json:"-"`
}

type GetTransctionsResponse struct{
	ID string `json:"id"`
	Amount float64 `json:"amount"`
	Type TransactionType `json:"type"`
	KWh float64 `gorm:"column:k_wh" json:"kwh"`
	CreatedAt string `json:"created_at"`
}


type TariffTier struct {
	BaseModel
	Limit float64 `gorm:"column:tier_limit;not null" json:"limit"`
	Rate  float64 `gorm:"column:tier_rate;not null"  json:"rate"`
}

func (t *TariffTier) Set() error {
	return db.DB.Create(t).Error
}

func GetTariffTiers() ([]TariffTier, error) {
	var tiers []TariffTier
	err := db.DB.Order("tier_limit asc").Find(&tiers).Error
	if err != nil {
		return nil, err
	}
	if len(tiers) == 0 {
		return nil, fmt.Errorf("no tariff tiers configured")
	}
	return tiers, nil
}

func DeleteTariffTierByID(id uuid.UUID) error {
	tx := db.DB.Where("id = ?", id).Delete(&TariffTier{})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}




func (w *Wallet) Create(tx *gorm.DB) error {
	if tx == nil {
		tx = db.DB
	}
	return tx.Create(w).Error
}

func (t *Transaction) Create(tx *gorm.DB) error {
	if tx == nil {
		tx = db.DB
	}
	return tx.Create(t).Error
}

func (w *Wallet) Save(tx *gorm.DB) error {
	if tx == nil {
		tx = db.DB
	}
	return tx.Save(w).Error
}

func GetWalletByUserID(userID uuid.UUID) (*Wallet, error) {
	var wallet Wallet
	err := db.DB.Where("user_id = ?", userID).First(&wallet).Error
	return &wallet, err
}

func GetAllTransaction() ([]GetTransctionsResponse, error) {
	var transactions []GetTransctionsResponse
	err := db.DB.Model(&Transaction{}).Select("id, amount, type, k_wh, created_at").Scan(&transactions).Error
	return transactions, err
}


func GetTransactionsByWalletID(walletID uuid.UUID) ([]GetTransctionsResponse, error) {
	var transactions []GetTransctionsResponse
	err := db.DB.Model(&Transaction{}).Where("wallet_id = ?", walletID).Select("id, amount, type, k_wh, created_at").Scan(&transactions).Error
	return transactions, err
}

func GetTransactionByTransactionID(transactionID uuid.UUID) (*Transaction, error) {
	var transaction Transaction
	err := db.DB.Preload("Wallet").Where("id = ?", transactionID).First(&transaction).Error
	return &transaction, err
}
 

 

func sortAndValidateTiers(tiers []TariffTier) ([]TariffTier, error) {
	sort.Slice(tiers, func(i, j int) bool {
		return tiers[i].Limit < tiers[j].Limit
	})

	for i := 1; i < len(tiers); i++ {
		if tiers[i].Limit == tiers[i-1].Limit {
			return nil, fmt.Errorf("duplicate tier limit %.2f with different rates (%.4f, %.4f)",
				tiers[i].Limit, tiers[i-1].Rate, tiers[i].Rate)
		}
	}

	return tiers, nil
}


func CalculateCost(kwh float64) (float64, error) {
	if kwh <= 0 {
		return 0, nil
	}

	raw, err := GetTariffTiers()
	if err != nil {
		return 0, err
	}

	tiers, err := sortAndValidateTiers(raw)
	if err != nil {
		return 0, err
	}

	var totalCost float64
	consumed := 0.0

	for i, tier := range tiers {
		if consumed >= kwh {
			break
		}

		usageInBand := min(kwh, tier.Limit) - consumed
		if usageInBand > 0 {
			totalCost += usageInBand * tier.Rate
			consumed += usageInBand
		}

		if i == len(tiers)-1 && kwh > tier.Limit {
			totalCost += (kwh - tier.Limit) * tier.Rate
		}
	}

	return totalCost, nil
}

func CalculatePower(amount float64) (float64, error) {
	if amount <= 0 {
		return 0, nil
	}

	raw, err := GetTariffTiers()
	if err != nil {
		return 0, err
	}

	tiers, err := sortAndValidateTiers(raw)
	if err != nil {
		return 0, err
	}

	var totalKWh float64
	remaining := amount

	for i, tier := range tiers {
		if remaining <= 0 {
			break
		}

		bandFloor := 0.0
		if i > 0 {
			bandFloor = tiers[i-1].Limit
		}

		bandCapacity := tier.Limit - bandFloor
		bandCost := bandCapacity * tier.Rate

		if remaining >= bandCost {
			totalKWh += bandCapacity
			remaining -= bandCost
		} else {
			totalKWh += remaining / tier.Rate
			remaining = 0
		}

		if i == len(tiers)-1 && remaining > 0 {
			totalKWh += remaining / tier.Rate
			remaining = 0
		}
	}

	return totalKWh, nil
}
