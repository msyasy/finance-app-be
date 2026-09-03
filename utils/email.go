package utils

import (
	"crypto/tls"
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

	if smtpHost == "" || smtpPort == "" || smtpEmail == "" || smtpPassword == "" {
		err := fmt.Errorf("variabel SMTP belum lengkap di environment")
		log.Println("[SMTP ERROR]:", err)
		return err
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", frontendURL, token)

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

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	log.Printf("[SMTP INFO] Mengirim email via SSL ke %s (%s)...", toEmail, addr)

	auth := smtp.PlainAuth("", smtpEmail, smtpPassword, smtpHost)

	// Dial menggunakan TLS/SSL langsung untuk port 465
	tlsconfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         smtpHost,
	}

	conn, err := tls.Dial("tcp", addr, tlsconfig)
	if err != nil {
		log.Println("[SMTP ERROR TLS Dial Failed]:", err)
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		log.Println("[SMTP ERROR NewClient Failed]:", err)
		return err
	}
	defer client.Quit()

	if err = client.Auth(auth); err != nil {
		log.Println("[SMTP ERROR Auth Failed]:", err)
		return err
	}

	if err = client.Mail(smtpEmail); err != nil {
		log.Println("[SMTP ERROR Mail From Failed]:", err)
		return err
	}

	if err = client.Rcpt(toEmail); err != nil {
		log.Println("[SMTP ERROR Rcpt To Failed]:", err)
		return err
	}

	w, err := client.Data()
	if err != nil {
		log.Println("[SMTP ERROR Data Failed]:", err)
		return err
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		log.Println("[SMTP ERROR Write Failed]:", err)
		return err
	}

	err = w.Close()
	if err != nil {
		log.Println("[SMTP ERROR Close Failed]:", err)
		return err
	}

	log.Println("[SMTP SUCCESS] Email reset password berhasil terkirim!")
	return nil
}