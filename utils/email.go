package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

type ResendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Html    string   `json:"html"`
}

func SendResetPasswordEmail(toEmail, token string) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	frontendURL := os.Getenv("FRONTEND_URL")

	if apiKey == "" {
		err := fmt.Errorf("RESEND_API_KEY belum dikonfigurasi di environment")
		log.Println("[RESEND ERROR]:", err)
		return err
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

	// Pengirim default untuk akun gratis Resend
	payload := ResendPayload{
		From:    "Aplikasi Keuangan <onboarding@resend.dev>",
		To:      []string{toEmail},
		Subject: "Reset Password - Aplikasi Keuangan",
		Html:    htmlBody,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Println("[RESEND ERROR Marshal]:", err)
		return err
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Println("[RESEND ERROR Request]:", err)
		return err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("[RESEND ERROR Send]:", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		log.Println("[RESEND ERROR API Response]:", errResp)
		return fmt.Errorf("gagal mengirim email via Resend API: %v", errResp)
	}

	log.Println("[RESEND SUCCESS] Email reset password berhasil terkirim ke", toEmail)
	return nil
}