package provider

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
)

type SMTPProvider struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSMTPProvider() *SMTPProvider {
	return &SMTPProvider{
		host:     getEnvOrDefault("SMTP_HOST", "smtp.mailjet.com"),
		port:     getEnvOrDefault("SMTP_PORT", "587"),
		username: os.Getenv("SMTP_USERNAME"),
		password: os.Getenv("SMTP_PASSWORD"),
		from:     getEnvOrDefault("SMTP_FROM", "noreply@example.com"),
	}
}

func (p *SMTPProvider) Send(_ context.Context, msg EmailMessage) error {
	addr := net.JoinHostPort(p.host, p.port)

	auth := smtp.PlainAuth("", p.username, p.password, p.host)

	tlsCfg := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         p.host,
	}

	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		plainConn, err2 := smtp.Dial(addr)
		if err2 != nil {
			return fmt.Errorf("smtp dial: %w", err2)
		}
		if err2 = plainConn.StartTLS(tlsCfg); err2 != nil {
			return fmt.Errorf("starttls: %w", err2)
		}
		if err2 = plainConn.Auth(auth); err2 != nil {
			return fmt.Errorf("smtp auth: %w", err2)
		}
		return sendViaSMTPClient(plainConn, p.from, msg)
	}

	client, err := smtp.NewClient(conn, p.host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	return sendViaSMTPClient(client, p.from, msg)
}

func sendViaSMTPClient(client *smtp.Client, from string, msg EmailMessage) error {
	defer client.Quit()

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp MAIL: %w", err)
	}
	if err := client.Rcpt(msg.To); err != nil {
		return fmt.Errorf("smtp RCPT: %w", err)
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	defer wc.Close()

	body := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		from, msg.To, msg.Subject, msg.Body)
	if _, err := fmt.Fprint(wc, body); err != nil {
		return fmt.Errorf("write email body: %w", err)
	}

	return nil
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
