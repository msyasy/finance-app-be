package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"finance-app-be/config"
	"finance-app-be/models"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/golang-jwt/jwt/v5"
)

func getWebAuthnHandler(c *gin.Context) (*webauthn.WebAuthn, error) {
	// 1. Cek jika WEBAUTHN_RP_ID diset manual di environment variable
	rpID := os.Getenv("WEBAUTHN_RP_ID")

	// 2. Jika tidak diset, ambil domain browser pengguna dari Header "Origin" atau "Referer"
	if rpID == "" {
		originHeader := c.Request.Header.Get("Origin")
		if originHeader == "" {
			originHeader = c.Request.Header.Get("Referer")
		}

		if originHeader != "" {
			if parsedURL, err := url.Parse(originHeader); err == nil && parsedURL.Hostname() != "" {
				rpID = parsedURL.Hostname()
			}
		}
	}

	// 3. Fallback ke Host request
	if rpID == "" {
		host := c.Request.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		rpID = host
	}

	origins := []string{
		"http://localhost:5173",
		"https://lapkeu.zone.id",
		"https://www.lapkeu.zone.id",
		"https://lapkeu-msyasy.vercel.app",
		"https://lapkeu.msyasy.xyz",
		"https://www.lapkeu.msyasy.xyz",
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL != "" {
		cleanURL := strings.TrimSuffix(frontendURL, "/")
		origins = append(origins, cleanURL)
	}

	if rpID != "" && rpID != "localhost" {
		origins = append(origins, "https://"+rpID)
		origins = append(origins, "http://"+rpID)
	}

	return webauthn.New(&webauthn.Config{
		RPDisplayName: "LapKeu Finance",
		RPID:          rpID,
		RPOrigins:     origins,
	})
}

type WebAuthnUser struct {
	ID          int
	Email       string
	Name        string
	Credentials []webauthn.Credential
}

func (u *WebAuthnUser) WebAuthnID() []byte {
	return []byte(strconv.Itoa(u.ID))
}

func (u *WebAuthnUser) WebAuthnName() string {
	return u.Email
}

func (u *WebAuthnUser) WebAuthnDisplayName() string {
	return u.Name
}

func (u *WebAuthnUser) WebAuthnIcon() string {
	return ""
}

func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

func getWebAuthnUserByID(userID int) (*WebAuthnUser, error) {
	var user models.User
	err := config.DB.QueryRow("SELECT id, name, email FROM users WHERE id = $1", userID).Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		return nil, err
	}

	creds := fetchUserCredentials(user.ID)

	return &WebAuthnUser{
		ID:          user.ID,
		Email:       user.Email,
		Name:        user.Name,
		Credentials: creds,
	}, nil
}

func getWebAuthnUserByEmail(email string) (*WebAuthnUser, error) {
	var user models.User
	err := config.DB.QueryRow("SELECT id, name, email FROM users WHERE email = $1", email).Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		return nil, err
	}

	creds := fetchUserCredentials(user.ID)

	return &WebAuthnUser{
		ID:          user.ID,
		Email:       user.Email,
		Name:        user.Name,
		Credentials: creds,
	}, nil
}

func fetchUserCredentials(userID int) []webauthn.Credential {
	rows, err := config.DB.Query(`
		SELECT credential_id, public_key, attestation_type, sign_count, user_present, user_verified, backup_eligible, backup_state
		FROM webauthn_credentials WHERE user_id = $1`, userID)
	if err != nil {
		log.Printf("[FETCH CREDENTIALS DB ERROR]: %v", err)
		return nil
	}
	defer rows.Close()

	creds := make([]webauthn.Credential, 0)
	for rows.Next() {
		var c webauthn.Credential
		var attType string
		var signCount int64
		err := rows.Scan(
			&c.ID,
			&c.PublicKey,
			&attType,
			&signCount,
			&c.Flags.UserPresent,
			&c.Flags.UserVerified,
			&c.Flags.BackupEligible,
			&c.Flags.BackupState,
		)
		if err != nil {
			log.Printf("[FETCH CREDENTIALS SCAN ERROR]: %v", err)
			continue
		}
		c.AttestationType = attType
		c.Authenticator.SignCount = uint32(signCount)
		creds = append(creds, c)
	}
	log.Printf("[FETCH CREDENTIALS SUCCESS]: Loaded %d credentials for userID %d", len(creds), userID)
	return creds
}

// 1. Begin Registration (POST /api/webauthn/register/begin) - Protected
func BeginRegistration(c *gin.Context) {
	wHandler, err := getWebAuthnHandler(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal inisialisasi WebAuthn: " + err.Error()})
		return
	}

	userID := getUserIDFromCategoryCtx(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User tidak terautentikasi"})
		return
	}

	user, err := getWebAuthnUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	options, sessionData, err := wHandler.BeginRegistration(user,
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationPreferred,
		}),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai registrasi biometrik: " + err.Error()})
		return
	}

	// Simpan sessionData ke DB
	sessBytes, _ := json.Marshal(sessionData)
	challengeKey := fmt.Sprintf("reg_%d", userID)
	_, _ = config.DB.Exec("DELETE FROM webauthn_sessions WHERE challenge_id = $1", challengeKey)
	_, err = config.DB.Exec("INSERT INTO webauthn_sessions (challenge_id, user_id, session_data, expires_at) VALUES ($1, $2, $3, $4)",
		challengeKey, userID, string(sessBytes), time.Now().Add(5*time.Minute))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan sesi registrasi"})
		return
	}

	c.JSON(http.StatusOK, options)
}

// 2. Finish Registration (POST /api/webauthn/register/finish) - Protected
func FinishRegistration(c *gin.Context) {
	wHandler, err := getWebAuthnHandler(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal inisialisasi WebAuthn"})
		return
	}

	userID := getUserIDFromCategoryCtx(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User tidak terautentikasi"})
		return
	}

	user, err := getWebAuthnUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	challengeKey := fmt.Sprintf("reg_%d", userID)
	var sessionJSON string
	err = config.DB.QueryRow("SELECT session_data FROM webauthn_sessions WHERE challenge_id = $1 AND user_id = $2", challengeKey, userID).Scan(&sessionJSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sesi registrasi tidak ditemukan atau sudah kadaluarsa"})
		return
	}

	var sessionData webauthn.SessionData
	if err := json.Unmarshal([]byte(sessionJSON), &sessionData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Format sesi tidak valid"})
		return
	}

	credential, err := wHandler.FinishRegistration(user, sessionData, c.Request)
	if err != nil {
		log.Printf("[WEBAUTHN FINISH REGISTRATION ERROR]: %v (User: %s)", err, user.Email)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Verifikasi biometrik gagal: " + err.Error()})
		return
	}

	// Simpan credential ke DB
	_, err = config.DB.Exec(`
		INSERT INTO webauthn_credentials (user_id, credential_id, public_key, attestation_type, sign_count, backup_eligible, backup_state)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (credential_id) DO UPDATE SET public_key = EXCLUDED.public_key, sign_count = EXCLUDED.sign_count`,
		userID, credential.ID, credential.PublicKey, credential.AttestationType, credential.Authenticator.SignCount, credential.Flags.BackupEligible, credential.Flags.BackupState)
	if err != nil {
		log.Printf("[WEBAUTHN DB SAVE ERROR]: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan data biometrik"})
		return
	}

	// Hapus sesi
	_, _ = config.DB.Exec("DELETE FROM webauthn_sessions WHERE challenge_id = $1", challengeKey)

	c.JSON(http.StatusOK, gin.H{"message": "Biometrik (Passkey) berhasil didaftarkan!"})
}

// 3. Begin Login (POST /api/webauthn/login/begin) - Public
func BeginLogin(c *gin.Context) {
	wHandler, err := getWebAuthnHandler(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal inisialisasi WebAuthn"})
		return
	}

	var input struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email wajib diisi untuk login biometrik"})
		return
	}

	user, err := getWebAuthnUserByEmail(input.Email)
	if err != nil || len(user.Credentials) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Akun ini belum mendaftarkan autentikasi biometrik"})
		return
	}

	options, sessionData, err := wHandler.BeginLogin(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai login biometrik: " + err.Error()})
		return
	}

	sessBytes, _ := json.Marshal(sessionData)
	challengeKey := fmt.Sprintf("log_%d", user.ID)
	_, _ = config.DB.Exec("DELETE FROM webauthn_sessions WHERE challenge_id = $1", challengeKey)
	_, err = config.DB.Exec("INSERT INTO webauthn_sessions (challenge_id, user_id, session_data, expires_at) VALUES ($1, $2, $3, $4)",
		challengeKey, user.ID, string(sessBytes), time.Now().Add(5*time.Minute))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan sesi login biometrik"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"options": options,
		"user_id": user.ID,
	})
}

// 4. Finish Login (POST /api/webauthn/login/finish) - Public
func FinishLogin(c *gin.Context) {
	wHandler, err := getWebAuthnHandler(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal inisialisasi WebAuthn"})
		return
	}

	userIDStr := c.Query("user_id")
	userID, _ := strconv.Atoi(userIDStr)
	if userID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID tidak valid"})
		return
	}

	user, err := getWebAuthnUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	challengeKey := fmt.Sprintf("log_%d", userID)
	var sessionJSON string
	err = config.DB.QueryRow("SELECT session_data FROM webauthn_sessions WHERE challenge_id = $1 AND user_id = $2", challengeKey, userID).Scan(&sessionJSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sesi login biometrik kadaluarsa"})
		return
	}

	var sessionData webauthn.SessionData
	if err := json.Unmarshal([]byte(sessionJSON), &sessionData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Format sesi tidak valid"})
		return
	}

	credential, err := wHandler.FinishRegistration(user, sessionData, c.Request)
	if err != nil {
		// Try FinishLogin as well
		credLogin, errLogin := wHandler.FinishLogin(user, sessionData, c.Request)
		if errLogin != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Verifikasi biometrik gagal: " + err.Error()})
			return
		}
		credential = credLogin
	}

	// Update sign count
	_, _ = config.DB.Exec("UPDATE webauthn_credentials SET sign_count = $1 WHERE credential_id = $2", credential.Authenticator.SignCount, credential.ID)
	_, _ = config.DB.Exec("DELETE FROM webauthn_sessions WHERE challenge_id = $1", challengeKey)

	// Buat JWT Token untuk login
	jwtSecretStr := os.Getenv("JWT_SECRET")
	if jwtSecretStr == "" {
		jwtSecretStr = "secretkeyrahasia"
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString([]byte(jwtSecretStr))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token login"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"user": gin.H{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		},
	})
}
