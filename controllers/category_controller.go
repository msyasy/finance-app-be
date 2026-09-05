package controllers

import (
	"finance-app-be/config"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type CategoryInput struct {
	Name string `json:"name" binding:"required"`
	Type string `json:"type" binding:"required"`
}

type BudgetInput struct {
	BudgetLimit float64 `json:"budget_limit" binding:"gte=0"`
}

func getUserIDFromCategoryCtx(c *gin.Context) int {
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

func CreateCategory(c *gin.Context) {
	var input CategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserIDFromCategoryCtx(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID tidak valid"})
		return
	}

	cleanName := strings.TrimSpace(input.Name)

	var count int
	err := config.DB.QueryRow("SELECT COUNT(*) FROM categories WHERE (user_id = $1 OR user_id IS NULL) AND LOWER(name) = LOWER($2) AND type = $3", userID, cleanName, input.Type).Scan(&count)
	if err == nil && count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kategori tersebut sudah ada"})
		return
	}

	var categoryID int
	query := "INSERT INTO categories (user_id, name, type) VALUES ($1, $2, $3) RETURNING id"
	err = config.DB.QueryRow(query, userID, cleanName, input.Type).Scan(&categoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat kategori baru"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Kategori berhasil dibuat", "category_id": categoryID})
}

func GetCategories(c *gin.Context) {
	userID := getUserIDFromCategoryCtx(c)

	var totalCount int
	_ = config.DB.QueryRow("SELECT COUNT(*) FROM categories").Scan(&totalCount)
	if totalCount == 0 {
		_, _ = config.DB.Exec("INSERT INTO categories (name, type) VALUES ('Gaji / Pendapatan', 'income'), ('Makanan & Minuman', 'expense'), ('Transportasi', 'expense'), ('Hiburan', 'expense')")
	}

	rows, err := config.DB.Query("SELECT id, name, COALESCE(type, 'expense') as type, COALESCE(budget_limit, 0) as budget_limit FROM categories WHERE user_id = $1 OR user_id IS NULL ORDER BY id ASC", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data kategori"})
		return
	}
	defer rows.Close()

	type CategoryRes struct {
		ID          int     `json:"id"`
		Name        string  `json:"name"`
		Type        string  `json:"type"`
		BudgetLimit float64 `json:"budget_limit"`
	}

	categories := make([]CategoryRes, 0)
	for rows.Next() {
		var cat CategoryRes
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Type, &cat.BudgetLimit); err == nil {
			categories = append(categories, cat)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": categories})
}

// SetCategoryBudget mengatur batas anggaran bulanan untuk kategori tertentu
func SetCategoryBudget(c *gin.Context) {
	categoryID := c.Param("id")
	userID := getUserIDFromCategoryCtx(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID tidak valid"})
		return
	}

	var input BudgetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nominal anggaran tidak valid"})
		return
	}

	res, err := config.DB.Exec("UPDATE categories SET budget_limit = $1 WHERE id = $2 AND (user_id = $3 OR user_id IS NULL)", input.BudgetLimit, categoryID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui batas anggaran"})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kategori tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Batas anggaran berhasil diperbarui"})
}