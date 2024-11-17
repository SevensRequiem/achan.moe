package mail

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"achan.moe/utils/queue"
	mail "github.com/wneessen/go-mail" // Ensure the correct version and alias
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

	msg := mail.NewMsg()
	if err := msg.From(smtpUser); err != nil {
		return fmt.Errorf("failed to set From address: %v", err)
	}
	if err := msg.To(m.To); err != nil {
		return fmt.Errorf("failed to set To address: %v", err)
	}
	msg.Subject(m.Subject)
	msg.SetBodyString(mail.TypeTextPlain, m.Body)

	client, err := mail.NewClient(smtpHost, mail.WithPort(port), mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(smtpUser), mail.WithPassword(smtpPass))
	if err != nil {
		return fmt.Errorf("failed to create mail client: %v", err)
	}

	if err := client.DialAndSend(msg); err != nil {
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

func AddMailToQueue(to, subject, body string) {
	queue.NewQueue().Enqueue("mail:send", func() {
		SendEmail(to, subject, body)
	})
}

func TestQueue() {
	AddMailToQueue("admin@requiem.moe", "Test Subject", "This is a test email.")
}
