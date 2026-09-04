package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type BrevoSender struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type BrevoRecipient struct {
	Email string `json:"email"`
}

type BrevoPayload struct {
	Sender      BrevoSender      `json:"sender"`
	To          []BrevoRecipient `json:"to"`
	Subject     string           `json:"subject"`
	HtmlContent string           `json:"htmlContent"`
}

func SendResetPasswordEmail(toEmail, token string) error {
	apiKey := os.Getenv("BREVO_API_KEY")
	senderEmail := os.Getenv("SMTP_EMAIL")
	frontendURL := os.Getenv("FRONTEND_URL")

	frontendURL = strings.TrimSpace(frontendURL)
	frontendURL = strings.Trim(frontendURL, "[]")
	if idx := strings.Index(frontendURL, "]("); idx != -1 {
		frontendURL = frontendURL[:idx]
		frontendURL = strings.TrimPrefix(frontendURL, "[")
	}

	if frontendURL == "" {
		frontendURL = "https://lapkeu.zone.id"
	}

	if apiKey == "" {
		err := fmt.Errorf("BREVO_API_KEY belum dikonfigurasi di environment variable")
		log.Println("[BREVO ERROR]:", err)
		return err
	}

	if senderEmail == "" {
		senderEmail = "msyasy.care@gmail.com"
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", frontendURL, token)

	htmlBody := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
			<h2>Permintaan Reset Password</h2>
			<p>Kamu menerima email ini karena ada permintaan untuk mereset password akun aplikasi keuangan kamu.</p>
			<p>Klik tombol di bawah ini untuk melanjutkan. Link ini hanya berlaku selama 15 menit:</p>
			<p style="margin: 25px 0;">
				<a href="%s" style="background-color: #2563eb; color: #ffffff; padding: 12px 20px; text-decoration: none; border-radius: 6px; font-weight: bold; display: inline-block;">Reset Password</a>
			</p>
			<p>Atau salin tautan berikut ke browser kamu:</p>
			<p><a href="%s">%s</a></p>
			<hr style="border: none; border-top: 1px solid #eee; margin-top: 30px;" />
			<p style="font-size: 12px; color: #888;">Jika kamu tidak meminta reset password, abaikan email ini.</p>
		</div>
	`, resetLink, resetLink, resetLink)

	payload := BrevoPayload{
		Sender:      BrevoSender{Name: "Lapkeu App", Email: senderEmail},
		To:          []BrevoRecipient{{Email: toEmail}},
		Subject:     "Reset Password - Lapkeu",
		HtmlContent: htmlBody,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://api.brevo.com/v3/smtp/email", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("[BREVO ERROR Send]:", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		log.Println("[BREVO API ERROR]:", errResp)
		return fmt.Errorf("gagal mengirim email via Brevo API: %v", errResp)
	}

	log.Println("[BREVO SUCCESS] Email reset password berhasil terkirim ke", toEmail)
	return nil
}