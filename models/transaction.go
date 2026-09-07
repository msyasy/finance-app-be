package models

import "time"

type Transaction struct {
	ID              int       `json:"id"`
	UserID          int       `json:"user_id"`
	WalletID        int       `json:"wallet_id"`
	CategoryID      int       `json:"category_id"`
	Amount          float64   `json:"amount"`
	Notes           string    `json:"notes"`
	TransactionDate time.Time `json:"transaction_date"`
	CreatedAt       time.Time `json:"created_at"`
}

type TransactionInput struct {
	WalletID   int     `json:"wallet_id" binding:"required"`
	CategoryID int     `json:"category_id" binding:"required"`
	Type       string  `json:"type" binding:"required"`
	Amount     float64 `json:"amount" binding:"required"`
	Notes      string  `json:"notes"`
	Note       string  `json:"note"`
}

func (t *TransactionInput) GetNotes() string {
	if t.Note != "" {
		return t.Note
	}
	return t.Notes
}