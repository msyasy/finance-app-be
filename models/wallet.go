package models

import "time"

type Wallet struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Name      string    `json:"name"`
	Balance   float64   `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
}

type WalletInput struct {
	Name    string  `json:"name" binding:"required"`
	Balance float64 `json:"balance"`
}