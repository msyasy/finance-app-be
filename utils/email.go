package utils

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"
)

func SendResetPasswordEmail(toEmail, token string) error {
	smtpHost := os.Getenv("SMTP_HOST")  
	smtpPort := os.Getenv("SMTP_PORT")
	smtpEmail := os.Getenv("SMTP_EMAIL")
	smtpPass := os.Getenv("SMTP_PASSWORD")
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

	if smtpHost == "" || smtpEmail == "" || smtpPass == "" {
		err := fmt.Errorf("konfigurasi SMTP belum lengkap di environment variable")
		log.Println("[SMTP ERROR]:", err)
		return err
	}

	if smtpPort == "" {
		smtpPort = "465"
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", frontendURL, token)

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

	// Koneksi SSL langsung via Port 465 dengan Timeout 10 Detik
	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         smtpHost,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	if err != nil {
		log.Println("[SMTP Dial Error]:", err)
		return fmt.Errorf("gagal terhubung ke server Gmail: %v", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		log.Println("[SMTP Client Error]:", err)
		return err
	}
	defer client.Quit()

	auth := smtp.PlainAuth("", smtpEmail, smtpPass, smtpHost)
	if err = client.Auth(auth); err != nil {
		log.Println("[SMTP Auth Error]:", err)
		return fmt.Errorf("autentikasi Gmail gagal: %v", err)
	}

	if err = client.Mail(smtpEmail); err != nil {
		return err
	}
	if err = client.Rcpt(toEmail); err != nil {
		return err
	}

	w, err := client.Data()
	if err != nil {
		return err
	}

	_, err = w.Write(msg)
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	log.Println("[SMTP SUCCESS] Email reset password berhasil terkirim ke", toEmail)
	return nil
}