package provider

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"time"
)

type SimulatedProvider struct {
	FailureRate float64
	LatencyMin  time.Duration
	LatencyMax  time.Duration
}

func NewSimulatedProvider() *SimulatedProvider {
	return &SimulatedProvider{
		FailureRate: 0.3,
		LatencyMin:  50 * time.Millisecond,
		LatencyMax:  200 * time.Millisecond,
	}
}

func (p *SimulatedProvider) Send(ctx context.Context, msg EmailMessage) error {
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

	if rand.Float64() < p.FailureRate {
		log.Printf("[SimulatedProvider] send failed for %s", msg.To)
		return errors.New("simulated provider error: transient network failure")
	}

	log.Printf("[SimulatedProvider] email sent to %s | subject: %s", msg.To, msg.Subject)
	return nil
}
