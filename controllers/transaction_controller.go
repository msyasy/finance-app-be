package controllers

import (
	"finance-app-be/config"
	"finance-app-be/models" // Import package models
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Helper Context User ID
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

// CREATE TRANSACTION
func CreateTransaction(c *gin.Context) {
	var input models.TransactionInput // Gunakan struct dari package models
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

	// Ambil teks catatan yang valid (baik dari "note" maupun "notes")
	finalNotes := input.GetNotes()

	// Insert Transaksi
	_, err = tx.Exec("INSERT INTO transactions (wallet_id, category_id, type, amount, notes) VALUES ($1, $2, $3, $4, $5)",
		input.WalletID, input.CategoryID, input.Type, input.Amount, finalNotes)
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