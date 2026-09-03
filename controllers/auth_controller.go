package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"finance-app-be/config"
	"finance-app-be/models"
	"finance-app-be/utils"
)

type RegisterInput struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal enkripsi password"})
		return
	}

	// Insert User Baru
	var userID int
	queryUser := "INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3) RETURNING id"
	err = config.DB.QueryRow(queryUser, input.Name, input.Email, string(hashedPassword)).Scan(&userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email sudah terdaftar atau gagal registrasi"})
		return
	}

	// Buat Dompet Utama Otomatis saat registrasi
	queryWallet := "INSERT INTO wallets (user_id, name, balance) VALUES ($1, $2, $3)"
	_, _ = config.DB.Exec(queryWallet, userID, "Dompet Utama", 0)

	c.JSON(http.StatusOK, gin.H{"message": "Registrasi berhasil"})
}

func Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	query := "SELECT id, name, email, password_hash FROM users WHERE email = $1"
	err := config.DB.QueryRow(query, input.Email).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email atau password salah"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email atau password salah"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		jwtSecret = []byte("secretkeyrahasia")
	}

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// --- LOGIKA LUPA & RESET PASSWORD ---

func generateRandomToken() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// POST /api/forgot-password
func ForgotPassword(c *gin.Context) {
	var input struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format email tidak valid"})
		return
	}

	var userId int
	// Cek apakah email terdaftar
	err := config.DB.QueryRow("SELECT id FROM users WHERE email = $1", input.Email).Scan(&userId)
	if err != nil {
		// Pesan sukses dikembalikan agar email terdaftar tidak mudah ditebak pihak luar
		c.JSON(http.StatusOK, gin.H{"message": "Jika email terdaftar, instruksi reset password telah dikirim ke email kamu."})
		return
	}

	token := generateRandomToken()
	expiresAt := time.Now().Add(15 * time.Minute)

	// Hapus token lama jika ada, lalu simpan token baru ke database
	_, _ = config.DB.Exec("DELETE FROM password_resets WHERE email = $1", input.Email)
	_, err = config.DB.Exec("INSERT INTO password_resets (email, token, expires_at) VALUES ($1, $2, $3)", input.Email, token, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses permintaan reset password"})
		return
	}

	// Kirim email lewat Goroutine (background process)
	go utils.SendResetPasswordEmail(input.Email, token)

	c.JSON(http.StatusOK, gin.H{"message": "Jika email terdaftar, instruksi reset password telah dikirim ke email kamu."})
}

// POST /api/reset-password
func ResetPassword(c *gin.Context) {
	var input struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token dan password baru (minimal 6 karakter) wajib diisi"})
		return
	}

	var email string
	var expiresAt time.Time
	err := config.DB.QueryRow("SELECT email, expires_at FROM password_resets WHERE token = $1", input.Token).Scan(&email, &expiresAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token reset tidak valid atau sudah pernah digunakan"})
		return
	}

	if time.Now().After(expiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token reset sudah kadaluarsa. Silakan minta link reset baru."})
		return
	}

	// Hash password baru
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses password baru"})
		return
	}

	// Update kolom password_hash di tabel users
	_, err = config.DB.Exec("UPDATE users SET password_hash = $1 WHERE email = $2", string(hashedPassword), email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mereset password"})
		return
	}

	// Hapus token yang sudah dipakai
	_, _ = config.DB.Exec("DELETE FROM password_resets WHERE token = $1", input.Token)

	c.JSON(http.StatusOK, gin.H{"message": "Password berhasil diperbarui! Silakan login dengan password baru."})
}