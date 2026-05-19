package provider

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"time"
)

// SimulatedProvider is the mock adapter that simulates a real email provider.
// It introduces realistic network latency and random transient failures
// so that retry/backoff logic can be properly tested.
type SimulatedProvider struct {
	// FailureRate is a value between 0 and 1 (e.g. 0.3 = 30% failure chance).
	FailureRate float64
	// LatencyMin/Max define the simulated network round-trip.
	LatencyMin time.Duration
	LatencyMax time.Duration
}

// NewSimulatedProvider creates a mock provider with sensible defaults.
func NewSimulatedProvider() *SimulatedProvider {
	return &SimulatedProvider{
		FailureRate: 0.3, // 30% random failure rate to test retry logic
		LatencyMin:  50 * time.Millisecond,
		LatencyMax:  200 * time.Millisecond,
	}
}

// Send simulates sending an email with network latency and random failures.
func (p *SimulatedProvider) Send(ctx context.Context, msg EmailMessage) error {
	// Simulate network latency
	latency := p.LatencyMin
	if p.LatencyMax > p.LatencyMin {
		spread := int64(p.LatencyMax - p.LatencyMin)
		latency = p.LatencyMin + time.Duration(rand.Int63n(spread))
	}

	select {
	case <-time.After(latency):
	case <-ctx.Done():
		return ctx.Err()
	}

	// Simulate random transient failure (e.g. SMTP timeout, provider down)
	if rand.Float64() < p.FailureRate {
		log.Printf("[SimulatedProvider] TRANSIENT FAILURE sending to %s (simulated)", msg.To)
		return errors.New("simulated provider error: transient network failure")
	}

	log.Printf("[SimulatedProvider] ✓ Email sent to %s | Subject: %s", msg.To, msg.Subject)
	return nil
}
