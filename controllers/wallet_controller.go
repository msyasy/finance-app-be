package controllers

import (
	"finance-app-be/config"
	"finance-app-be/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type WalletInput struct {
	Name    string  `json:"name" binding:"required"`
	Balance float64 `json:"balance"`
}

type TransferInput struct {
	SourceWalletID      int     `json:"source_wallet_id" binding:"required"`
	DestinationWalletID int     `json:"destination_wallet_id" binding:"required"`
	Amount              float64 `json:"amount" binding:"required,gt=0"`
}

func getUserIDFromWalletCtx(c *gin.Context) int {
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

func CreateWallet(c *gin.Context) {
	var input WalletInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserIDFromWalletCtx(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID tidak valid"})
		return
	}

	cleanName := strings.TrimSpace(input.Name)

	var count int
	err := config.DB.QueryRow("SELECT COUNT(*) FROM wallets WHERE user_id = $1 AND LOWER(name) = LOWER($2)", userID, cleanName).Scan(&count)
	if err == nil && count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dompet dengan nama tersebut sudah ada"})
		return
	}

	var walletID int
	query := "INSERT INTO wallets (user_id, name, balance) VALUES ($1, $2, $3) RETURNING id"
	err = config.DB.QueryRow(query, userID, cleanName, input.Balance).Scan(&walletID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan dompet ke database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Dompet berhasil dibuat", "wallet_id": walletID})
}

func GetWallets(c *gin.Context) {
	userID := getUserIDFromWalletCtx(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID tidak valid"})
		return
	}

	rows, err := config.DB.Query("SELECT id, user_id, name, balance FROM wallets WHERE user_id = $1 ORDER BY id ASC", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data dompet"})
		return
	}
	defer rows.Close()

	wallets := make([]models.Wallet, 0)
	for rows.Next() {
		var w models.Wallet
		if err := rows.Scan(&w.ID, &w.UserID, &w.Name, &w.Balance); err == nil {
			wallets = append(wallets, w)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": wallets})
}

func DeleteWallet(c *gin.Context) {
	walletID := c.Param("id")
	userID := getUserIDFromWalletCtx(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID tidak valid"})
		return
	}

	tx, err := config.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi DB"})
		return
	}

	_, _ = tx.Exec("DELETE FROM transactions WHERE wallet_id = $1", walletID)

	res, err := tx.Exec("DELETE FROM wallets WHERE id = $1 AND user_id = $2", walletID, userID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus dompet"})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Dompet tidak ditemukan atau akses ditolak"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Dompet berhasil dihapus"})
}

func TransferWallet(c *gin.Context) {
	var input TransferInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Input tidak valid"})
		return
	}

	userID := getUserIDFromWalletCtx(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID tidak valid"})
		return
	}

	if input.SourceWalletID == input.DestinationWalletID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dompet asal dan tujuan tidak boleh sama"})
		return
	}

	tx, err := config.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi DB"})
		return
	}

	var sourceBalance float64
	err = tx.QueryRow("SELECT balance FROM wallets WHERE id = $1 AND user_id = $2", input.SourceWalletID, userID).Scan(&sourceBalance)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Dompet asal tidak ditemukan"})
		return
	}

	if sourceBalance < input.Amount {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Saldo dompet asal tidak mencukupi"})
		return
	}

	var destBalance float64
	err = tx.QueryRow("SELECT balance FROM wallets WHERE id = $1 AND user_id = $2", input.DestinationWalletID, userID).Scan(&destBalance)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Dompet tujuan tidak ditemukan"})
		return
	}

	_, err = tx.Exec("UPDATE wallets SET balance = balance - $1 WHERE id = $2 AND user_id = $3", input.Amount, input.SourceWalletID, userID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memotong saldo dompet asal"})
		return
	}

	_, err = tx.Exec("UPDATE wallets SET balance = balance + $1 WHERE id = $2 AND user_id = $3", input.Amount, input.DestinationWalletID, userID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambah saldo dompet tujuan"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyelesaikan transfer"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transfer saldo berhasil"})
}