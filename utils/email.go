package utils

import (
	"fmt"
	"net/smtp"
	"os"
)

func SendResetPasswordEmail(toEmail, token string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpEmail := os.Getenv("SMTP_EMAIL")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	frontendURL := os.Getenv("FRONTEND_URL")

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", frontendURL, token)

	auth := smtp.PlainAuth("", smtpEmail, smtpPassword, smtpHost)

	subject := "Subject: Reset Password - Aplikasi Keuangan\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
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

	msg := []byte(subject + mime + body)
	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	return smtp.SendMail(addr, auth, smtpEmail, []string{toEmail}, msg)
}