package adapter

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"
)

// EmailSender is the Adapter Pattern interface.
// Use cases depend on this interface, not on concrete providers.
// This decouples business logic from vendor implementations.
type EmailSender interface {
	Send(ctx context.Context, to, subject, body string) error
}

// ─── Simulated Provider ──────────────────────────────────────────────────────

// SimulatedEmailSender logs the action, adds network latency, and
// occasionally fails randomly — used to test retry logic.
type SimulatedEmailSender struct {
	FailRate float64 // e.g. 0.3 = 30% failure rate
	Latency  time.Duration
}

func NewSimulatedSender(failRate float64, latency time.Duration) EmailSender {
	return &SimulatedEmailSender{FailRate: failRate, Latency: latency}
}

func (s *SimulatedEmailSender) Send(_ context.Context, to, subject, body string) error {
	// Simulate network latency
	time.Sleep(s.Latency)

	// Simulate random failure
	//nolint:gosec
	if rand.Float64() < s.FailRate {
		return fmt.Errorf("simulated provider failure: transient network error")
	}

	log.Printf("[SIMULATED EMAIL] → To: %s | Subject: %s | Body: %s", to, subject, body)
	return nil
}

// ─── SMTP Provider ───────────────────────────────────────────────────────────

// SMTPEmailSender is a real SMTP-based provider (Option A).
// Configure via env: SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS.
type SMTPEmailSender struct {
	Host     string
	Port     string
	Username string
	Password string
}

func NewSMTPSender(host, port, user, pass string) EmailSender {
	return &SMTPEmailSender{Host: host, Port: port, Username: user, Password: pass}
}

func (s *SMTPEmailSender) Send(_ context.Context, to, subject, body string) error {
	// In a real implementation this would use net/smtp or a library.
	// For this assignment we demonstrate the adapter boundary.
	log.Printf("[SMTP EMAIL] Host: %s | To: %s | Subject: %s", s.Host, to, subject)
	log.Printf("[SMTP EMAIL] Body: %s", body)
	return nil
}
