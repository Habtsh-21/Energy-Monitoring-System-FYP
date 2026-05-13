package services

import (
	"energy-monitoring-system/internal/db"
	"energy-monitoring-system/internal/models"
	"fmt"
	"math"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func round(v float64) float64 {
	return math.Round(v*100) / 100
}

func UpdateWallet(userID uuid.UUID, amountKwh float64, txType models.TransactionType,reference string)  error {
	if amountKwh <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	} 

	return db.DB.Transaction(func(tx *gorm.DB) error {
		var wallet models.Wallet
		if err := tx.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// Create wallet if it doesn't exist
				wallet = models.Wallet{UserID: userID, BalanceKwh: 0}
				if err := wallet.Create(tx); err != nil {
					return err
				}
			} else {
				return err
			}
		}


		balanceBefore := wallet.BalanceKwh
		if txType == models.TxTypeTopUp {
			wallet.BalanceKwh = round(wallet.BalanceKwh + amountKwh)
		} else {
			wallet.BalanceKwh = round(wallet.BalanceKwh - amountKwh)
		}
		if err := wallet.Save(tx); err != nil {
			return err
		}

		trx := models.Transaction{
			WalletID:       wallet.ID,
			Amount:         amountKwh,
			Type:           txType,
			Reference:      reference,
			Note:           "Top-up",
			BalanceBefore:  balanceBefore,
			BalanceAfter:   wallet.BalanceKwh,
			
		}
		if err := trx.Create(tx); err != nil {
			return err
		}

		

		return nil
	})
}
