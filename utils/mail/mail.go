package mail

import (
	"fmt"
	"log"
	"os"
	"strconv"

	gomail "gopkg.in/mail.v2"
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

	msg := gomail.NewMessage()
	msg.SetHeader("From", smtpUser)
	msg.SetHeader("To", m.To)
	msg.SetHeader("Subject", m.Subject)
	msg.SetBody("text/plain", m.Body)

	d := gomail.NewDialer(smtpHost, port, smtpUser, smtpPass)

	if err := d.DialAndSend(msg); err != nil {
		log.Printf("Failed to send email to %s: %v", m.To, err)
		return err
	}

	log.Printf("Email sent to %s", m.To)
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
