package controllers

import (
	"finance-app-be/config"
	"net/http"

	"github.com/gin-gonic/gin"
)

type TransactionInput struct {
	WalletID   int     `json:"wallet_id" binding:"required"`
	CategoryID int     `json:"category_id" binding:"required"`
	Type       string  `json:"type" binding:"required"` // "income" atau "expense"
	Amount     float64 `json:"amount" binding:"required"`
	Notes      string  `json:"notes"`
}

func getUserIDFromTxCtx(c *gin.Context) int {
	val, exists := c.Get("userID")
	if !exists {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	default:
		return 0
	}
}

func CreateTransaction(c *gin.Context) {
	var input TransactionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserIDFromTxCtx(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID tidak valid"})
		return
	}

	// Cek Saldo & Kepemilikan Wallet
	var currentBalance float64
	err := config.DB.QueryRow("SELECT balance FROM wallets WHERE id = $1 AND user_id = $2", input.WalletID, userID).Scan(&currentBalance)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dompet tidak ditemukan"})
		return
	}

	// Hitung Saldo Baru
	var newBalance float64
	if input.Type == "income" {
		newBalance = currentBalance + input.Amount
	} else if input.Type == "expense" {
		if currentBalance < input.Amount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Saldo dompet tidak mencukupi"})
			return
		}
		newBalance = currentBalance - input.Amount
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tipe transaksi tidak valid"})
		return
	}

	tx, err := config.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi DB"})
		return
	}

	// Insert Transaksi
	_, err = tx.Exec("INSERT INTO transactions (wallet_id, category_id, type, amount, notes) VALUES ($1, $2, $3, $4, $5)",
		input.WalletID, input.CategoryID, input.Type, input.Amount, input.Notes)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mencatat transaksi"})
		return
	}

	// Update Saldo Wallet
	_, err = tx.Exec("UPDATE wallets SET balance = $1 WHERE id = $2", newBalance, input.WalletID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui saldo"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Transaksi berhasil dicatat"})
}

func DeleteTransaction(c *gin.Context) {
	id := c.Param("id")
	userID := getUserIDFromTxCtx(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID tidak valid"})
		return
	}

	var walletID int
	var amount float64
	var txType string

	err := config.DB.QueryRow("SELECT wallet_id, COALESCE(type, 'expense'), amount FROM transactions WHERE id = $1", id).
		Scan(&walletID, &txType, &amount)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaksi tidak ditemukan"})
		return
	}

	var currentBalance float64
	err = config.DB.QueryRow("SELECT balance FROM wallets WHERE id = $1 AND user_id = $2", walletID, userID).
		Scan(&currentBalance)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Akses ditolak"})
		return
	}

	var newBalance float64
	if txType == "income" {
		newBalance = currentBalance - amount
	} else {
		newBalance = currentBalance + amount
	}

	tx, err := config.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal transaksi DB"})
		return
	}

	_, err = tx.Exec("DELETE FROM transactions WHERE id = $1", id)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus transaksi"})
		return
	}

	_, err = tx.Exec("UPDATE wallets SET balance = $1 WHERE id = $2", newBalance, walletID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate saldo"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Transaksi berhasil dihapus"})
}

func GetTransactions(c *gin.Context) {
	userID := getUserIDFromTxCtx(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID tidak valid"})
		return
	}

	rows, err := config.DB.Query(`
		SELECT t.id, t.wallet_id, t.category_id, COALESCE(t.type, 'expense') as type, t.amount, t.notes, t.created_at 
		FROM transactions t 
		JOIN wallets w ON w.id = t.wallet_id 
		WHERE w.user_id = $1 
		ORDER BY t.created_at DESC`, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data transaksi"})
		return
	}
	defer rows.Close()

	type TransactionRes struct {
		ID         int     `json:"id"`
		WalletID   int     `json:"wallet_id"`
		CategoryID int     `json:"category_id"`
		Type       string  `json:"type"`
		Amount     float64 `json:"amount"`
		Notes      string  `json:"notes"`
		CreatedAt  string  `json:"created_at"`
	}

	transactions := make([]TransactionRes, 0)
	for rows.Next() {
		var t TransactionRes
		if err := rows.Scan(&t.ID, &t.WalletID, &t.CategoryID, &t.Type, &t.Amount, &t.Notes, &t.CreatedAt); err == nil {
			transactions = append(transactions, t)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": transactions})
}