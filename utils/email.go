package utils

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strings"
)

func SendResetPasswordEmail(toEmail, token string) error {
	smtpHost := os.Getenv("SMTP_HOST")     // smtp.gmail.com
	smtpPort := os.Getenv("SMTP_PORT")     // 587
	smtpEmail := os.Getenv("SMTP_EMAIL")   // msyasy.care@gmail.com
	smtpPass := os.Getenv("SMTP_PASSWORD") // App Password
	frontendURL := os.Getenv("FRONTEND_URL")

	// Membersihkan format markdown jika tak sengaja tersimpan di env variable Railway
	frontendURL = strings.TrimSpace(frontendURL)
	frontendURL = strings.Trim(frontendURL, "[]")
	if idx := strings.Index(frontendURL, "]("); idx != -1 {
		frontendURL = frontendURL[:idx]
		frontendURL = strings.TrimPrefix(frontendURL, "[")
	}

	if frontendURL == "" {
		frontendURL = "https://lapkeu.zone.id"
	}

	if smtpHost == "" || smtpEmail == "" || smtpPass == "" {
		err := fmt.Errorf("konfigurasi SMTP belum lengkap di environment variable")
		log.Println("[SMTP ERROR]:", err)
		return err
	}

	if smtpPort == "" {
		smtpPort = "587"
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", frontendURL, token)

	// Format Header & Body Email HTML
	subject := "Subject: Reset Password - Lapkeu\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
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

	msg := []byte(subject + mime + htmlBody)

	// Autentikasi dan Pengiriman via Gmail SMTP
	auth := smtp.PlainAuth("", smtpEmail, smtpPass, smtpHost)
	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	err := smtp.SendMail(addr, auth, smtpEmail, []string{toEmail}, msg)
	if err != nil {
		log.Println("[SMTP ERROR Send]:", err)
		return fmt.Errorf("gagal mengirim email via SMTP: %v", err)
	}

	log.Println("[SMTP SUCCESS] Email reset password berhasil terkirim ke", toEmail)
	return nil
}