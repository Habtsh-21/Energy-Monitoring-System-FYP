package services

import (
	"energy-monitoring-system/internal/models"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrInsufficientBalance is returned when a debit would take the wallet below zero.
var ErrInsufficientBalance = errors.New("insufficient balance")

// ErrUserInactive is returned when a wallet operation is not allowed for an inactive account.
var ErrUserInactive = errors.New("user account is inactive")

func round(v float64) float64 {
	return math.Round(v*100) / 100
}

// DebitWallet deducts kwh from the user's wallet inside the provided transaction.
// Returns ErrInsufficientBalance if the wallet does not have enough credit.
func DebitWallet(tx *gorm.DB, userID uuid.UUID, kwh float64, reference string) error {
	if kwh <= 0 {
		return fmt.Errorf("kwh must be greater than 0")
	}

	var wallet models.Wallet
	if err := tx.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return fmt.Errorf("wallet lookup failed: %w", err)
	}

	if wallet.BalanceKwh < kwh {
		return ErrInsufficientBalance
	}

	balanceBefore := wallet.BalanceKwh
	wallet.BalanceKwh = round(wallet.BalanceKwh - kwh)

	if err := wallet.Save(tx); err != nil {
		return fmt.Errorf("failed to save wallet: %w", err)
	}

	trx := models.Transaction{
		WalletID:      wallet.ID,
		Amount:        kwh,
		KWh:           kwh,
		Type:          models.TxTypeUsageDebit,
		Reference:     reference,
		Note:          "Usage debit",
		BalanceBefore: balanceBefore,
		BalanceAfter:  wallet.BalanceKwh,
	}
	if err := trx.Create(tx); err != nil {
		return fmt.Errorf("failed to record transaction: %w", err)
	}

	return nil
}

// TopUpWallet credits kwh to the user's wallet. Creates the wallet if missing.
func TopUpWallet(tx *gorm.DB, userID uuid.UUID, kwh float64, reference string) error {
	if kwh <= 0 {
		return fmt.Errorf("kwh must be greater than 0")
	}

	var u models.User
	if err := tx.Where("id = ?", userID).First(&u).Error; err != nil {
		return fmt.Errorf("user lookup failed: %w", err)
	}
	if !u.IsActive {
		return ErrUserInactive
	}

	var wallet models.Wallet
	err := tx.Where("user_id = ?", userID).First(&wallet).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			wallet = models.Wallet{UserID: userID, BalanceKwh: 0}
			if err := wallet.Create(tx); err != nil {
				return fmt.Errorf("failed to create wallet: %w", err)
			}
		} else {
			return fmt.Errorf("wallet lookup failed: %w", err)
		}
	}

	balanceBefore := wallet.BalanceKwh
	wallet.BalanceKwh = round(wallet.BalanceKwh + kwh)

	if err := wallet.Save(tx); err != nil {
		return fmt.Errorf("failed to save wallet: %w", err)
	}

	trx := models.Transaction{
		WalletID:      wallet.ID,
		Amount:        kwh,
		KWh:           kwh,
		Type:          models.TxTypeTopUp,
		Reference:     reference,
		Note:          "Top-up",
		BalanceBefore: balanceBefore,
		BalanceAfter:  wallet.BalanceKwh,
	}
	if err := trx.Create(tx); err != nil {
		return fmt.Errorf("failed to record transaction: %w", err)
	}

	return nil
}
