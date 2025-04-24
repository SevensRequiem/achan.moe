package mail

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"strconv"

	"achan.moe/logs"
	"achan.moe/utils/queue"
	"github.com/go-mail/mail/v2"
)

type Mail struct {
	To      string
	Subject string
	Body    string
}

func (m *Mail) Send() error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_EMAIL")
	smtpPass := os.Getenv("SMTP_PASSWORD")

	if smtpHost == "" || smtpPort == "" || smtpUser == "" || smtpPass == "" {
		return fmt.Errorf("SMTP configuration is missing")
	}

	port, err := strconv.Atoi(smtpPort)
	if err != nil {
		return fmt.Errorf("Invalid SMTP port: %v", err)
	}

	d := mail.NewDialer(smtpHost, port, smtpUser, smtpPass)
	d.StartTLSPolicy = mail.NoStartTLS

	if d.TLSConfig == nil {
		d.TLSConfig = &tls.Config{}
	}

	d.TLSConfig.InsecureSkipVerify = true
	d.TLSConfig.ServerName = smtpHost
	d.TLSConfig.MinVersion = 0x0303 // TLS 1.2
	d.TLSConfig.MaxVersion = 0x0304 // TLS 1.3

	mail := mail.NewMessage()
	mail.SetHeader("From", smtpUser)
	mail.SetHeader("To", m.To)
	mail.SetHeader("Subject", m.Subject)
	mail.SetBody("text/html", m.Body)
	if err := d.DialAndSend(mail); err != nil {
		logs.Error("Failed to send email: %v", err)
		return fmt.Errorf("failed to send email: %w", err)
	}
	logs.Info("Email sent to %s with subject %s", m.To, m.Subject)
	return nil
}

func SendEmail(to, subject, body string) error {
	m := Mail{
		To:      to,
		Subject: subject,
		Body:    body,
	}
	return m.Send()
}

func TestMail() {
	err := SendEmail("admin@requiem.moe", "Test Subject", "This is a test email.")
	if err != nil {
		log.Fatalf("Error sending test email: %v", err)
	}
}

func AddMailToQueue(to, subject, body string) {
	queue.Q.Enqueue("mail:send", func() {
		SendEmail(to, subject, body)
	})
}

func TestQueue() {
	AddMailToQueue("admin@requiem.moe", "Test Subject", "This is a test email.")
}
