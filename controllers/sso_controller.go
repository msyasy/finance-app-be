package controllers

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"finance-app-be/config"
)

// ---------------------------------------------------------------------------
// issueJWT: satu-satunya tempat membuat token.
// Pakai juga di Login() menggantikan blok pembuatan token yang lama,
// supaya token hasil login biasa, passkey, dan SSO sama persis.
// Tidak ada secret cadangan: kalau JWT_SECRET kosong, gagal.
// ---------------------------------------------------------------------------
func issueJWT(userID int) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET belum di-set")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	return token.SignedString([]byte(secret))
}

// ---------------------------------------------------------------------------
// Identitas yang dikembalikan lowkit setelah kode ditukar.
// ---------------------------------------------------------------------------
type lowkitIdentity struct {
	UserID        string `json:"user_id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	EmailVerified bool   `json:"email_verified"`
}

var ssoHTTPClient = &http.Client{Timeout: 8 * time.Second}

var errSSOEmailConflict = errors.New("email sudah terdaftar di LapKeu dan tidak bisa ditautkan otomatis")

// exchangeCodeWithLowkit memanggil lowkit_be (server ke server).
// ENV yang dibutuhkan:
//   LOWKIT_API_URL     contoh: https://api.lowkit.my.id (tanpa slash di akhir)
//   LOWKIT_SSO_SECRET  string acak panjang, sama dengan yang dipasang di lowkit_be
func exchangeCodeWithLowkit(code string) (*lowkitIdentity, error) {
	base := os.Getenv("LOWKIT_API_URL")
	secret := os.Getenv("LOWKIT_SSO_SECRET")
	if base == "" || secret == "" {
		return nil, errors.New("konfigurasi SSO belum lengkap")
	}

	body, _ := json.Marshal(map[string]string{"code": code, "client_id": "lapkeu"})
	req, err := http.NewRequest(http.MethodPost, base+"/api/v1/internal/sso/exchange", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-SSO-Client-Secret", secret)

	resp, err := ssoHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lowkit menolak kode (status %d)", resp.StatusCode)
	}

	var id lowkitIdentity
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&id); err != nil {
		return nil, err
	}
	if id.UserID == "" || id.Email == "" {
		return nil, errors.New("respons lowkit tidak lengkap")
	}
	return &id, nil
}

// Password acak yang tidak diketahui siapa pun. User SSO baru tidak punya
// password sungguhan, tapi bisa membuatnya lewat "Lupa Password".
func randomPasswordHash() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	h, err := bcrypt.GenerateFromPassword(b, bcrypt.DefaultCost)
	return string(h), err
}

// findOrCreateSSOUser mengembalikan user LapKeu untuk identitas lowkit tersebut.
//  1. Sudah tertaut lewat lowkit_user_id -> pakai.
//  2. Email sama persis dan email di lowkit TERVERIFIKASI -> tautkan akun lama.
//  3. Belum ada -> buat akun baru beserta Dompet Utama.
// Email yang belum terverifikasi tidak pernah dipakai untuk menautkan akun,
// supaya orang lain tidak bisa mengambil alih akun keuangan dengan mendaftar
// memakai email korban di lowkit.
func findOrCreateSSOUser(id *lowkitIdentity) (userID int, name, email string, err error) {
	// 1. Sudah tertaut
	err = config.DB.QueryRow(
		"SELECT id, name, email FROM users WHERE lowkit_user_id = $1", id.UserID,
	).Scan(&userID, &name, &email)
	if err == nil {
		return userID, name, email, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, "", "", err
	}

	// 2. Tautkan akun lama berdasarkan email (hanya jika terverifikasi)
	if id.EmailVerified {
		err = config.DB.QueryRow(
			`UPDATE users SET lowkit_user_id = $1
			 WHERE lower(email) = lower($2) AND lowkit_user_id IS NULL
			 RETURNING id, name, email`,
			id.UserID, id.Email,
		).Scan(&userID, &name, &email)
		if err == nil {
			return userID, name, email, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, "", "", err
		}
	}

	// 3. Buat akun baru
	hash, err := randomPasswordHash()
	if err != nil {
		return 0, "", "", err
	}
	name = id.Name
	if name == "" {
		name = id.Email
	}
	err = config.DB.QueryRow(
		`INSERT INTO users (name, email, password_hash, lowkit_user_id)
		 VALUES ($1, $2, $3, $4) RETURNING id, name, email`,
		name, id.Email, hash, id.UserID,
	).Scan(&userID, &name, &email)
	if err != nil {
		// Biasanya karena email sudah dipakai akun LapKeu lain yang tidak boleh ditautkan otomatis.
		return 0, "", "", errSSOEmailConflict
	}

	if _, werr := config.DB.Exec(
		"INSERT INTO wallets (user_id, name, balance) VALUES ($1, $2, $3)",
		userID, "Dompet Utama", 0,
	); werr != nil {
		log.Println("[SSO WARNING]: Gagal membuat dompet utama ->", werr)
	}
	return userID, name, email, nil
}

// POST /api/sso/exchange   (publik, pakai strictAuthLimiter)
// Body: {"code": "..."}
func SSOExchange(c *gin.Context) {
	var input struct {
		Code string `json:"code" binding:"required,min=20,max=200"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kode login tidak valid"})
		return
	}

	identity, err := exchangeCodeWithLowkit(input.Code)
	if err != nil {
		log.Println("[SSO ERROR]: tukar kode ->", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Kode login tidak valid atau sudah kedaluwarsa. Silakan coba lagi."})
		return
	}

	userID, name, email, err := findOrCreateSSOUser(identity)
	if err != nil {
		if errors.Is(err, errSSOEmailConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "Email ini sudah terdaftar di LapKeu. Masuk dengan email dan password LapKeu, atau verifikasi email di LowonganKita."})
			return
		}
		log.Println("[SSO ERROR]: user ->", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses akun"})
		return
	}

	token, err := issueJWT(userID)
	if err != nil {
		log.Println("[SSO ERROR]: token ->", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat sesi login"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  gin.H{"id": userID, "name": name, "email": email},
	})
}