package utils

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
)

func SendResetPasswordEmail(toEmail, token string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpEmail := os.Getenv("SMTP_EMAIL")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	frontendURL := os.Getenv("FRONTEND_URL")

	// 1. Validasi variabel environment
	if smtpHost == "" || smtpPort == "" || smtpEmail == "" || smtpPassword == "" {
		err := fmt.Errorf("variabel SMTP belum lengkap di environment")
		log.Println("[SMTP ERROR]:", err)
		return err
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", frontendURL, token)

	// 2. Format Header & Content Email (Lengkap dengan From dan To)
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("Aplikasi Keuangan <%s>", smtpEmail)
	headers["To"] = toEmail
	headers["Subject"] = "Reset Password - Aplikasi Keuangan"
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"UTF-8\""

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}

	body := fmt.Sprintf(`
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

	message += "\r\n" + body

	// 3. Autentikasi dan Pengiriman Email
	auth := smtp.PlainAuth("", smtpEmail, smtpPassword, smtpHost)
	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	log.Printf("[SMTP INFO] Mengirim email reset password ke %s via %s...", toEmail, addr)

	err := smtp.SendMail(addr, auth, smtpEmail, []string{toEmail}, []byte(message))
	if err != nil {
		log.Println("[SMTP ERROR Gagal Kirim Email]:", err)
		return err
	}

	log.Println("[SMTP SUCCESS] Email reset password berhasil terkirim!")
	return nil
}