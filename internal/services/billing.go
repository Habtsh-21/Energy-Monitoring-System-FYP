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

func TopUpWallet(userID uuid.UUID, amount float64, reference string)  error {
	if amount <= 0 {
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


		purchasedkwh ,err := models.CalculatePower(amount);
		if err!= nil {
			return err
		}

		balanceBefore := wallet.BalanceKwh
		wallet.BalanceKwh = round(wallet.BalanceKwh + purchasedkwh)
		if err := wallet.Save(tx); err != nil {
			return err
		}

		trx := models.Transaction{
			WalletID:       wallet.ID,
			Amount:         round(amount),
			Type:           models.TxTypeTopUp,
			Reference:      reference,
			Note:           "Top-up",
			BalanceBefore:  balanceBefore,
			BalanceAfter:   wallet.BalanceKwh,
			
		}
		if err := trx.Create(tx); err != nil {
			return err
		}

		if wallet.BalanceKwh > 0 {
			var user models.User
			if err := tx.Where("id = ?", userID).First(&user).Error; err == nil && user.MeterID != uuid.Nil {
				var meter models.Meter
				if err := tx.Where("id = ?", user.MeterID).First(&meter).Error; err == nil {
					if meter.RelayStatus == "OFF" {
						meter.RelayStatus = "ON"
						tx.Save(&meter)
					}
				}
			}
		}

		return nil
	})
}
