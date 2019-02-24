package main

import (
	"fmt"
	"time"
)

type PulsingOutput struct {
	DigitalOutputPin
	v chan<- bool
}

// WriteBool is the only overridden method - redirect the
// writes through the channel to the goroutine to serialise
// them and condition the pulses appropriately
func (p *PulsingOutput) WriteBool(v bool) error {
	p.v <- v
	return nil
}

const (
	minAcceptablePulse = time.Millisecond * 5
	maxAcceptablePulse = time.Second * 300
)

func NewPulsingOutput(p DigitalOutputPin, pulse string) (DigitalOutputPin, error) {
	duration, err := time.ParseDuration(pulse)
	if err != nil {
		return nil, err
	}

	if duration < minAcceptablePulse {
		return nil, fmt.Errorf("Duration too short (%v), minimum %v", duration, minAcceptablePulse)
	}

	if duration > maxAcceptablePulse {
		return nil, fmt.Errorf("Duration too long (%v), maximum %v", duration, maxAcceptablePulse)
	}

	vchan := make(chan bool)
	go pulseControl(p, duration, vchan)

	return &PulsingOutput{p, vchan}, nil
}

func pulseControl(p DigitalOutputPin, d time.Duration, v <-chan bool) {
	// Create a stopped timer, ready to use when a pulse starts
	t := time.NewTimer(d)
	if !t.Stop() {
		<-t.C
	}
	// running allows us to determine whether the timer
	// should be stopped before resetting, to prevent a race
	running := false

	for {
		select {
		case value := <-v: // New output value to set
			// Discard error :-(
			p.WriteBool(value)

			if running && !t.Stop() {
				<-t.C
			}

			if value {
				t.Reset(d)
				running = true
			} else {
				running = false
			}

		case <-t.C: // Pulse time has expired
			p.WriteBool(false)
			running = false
		}
	}
}
