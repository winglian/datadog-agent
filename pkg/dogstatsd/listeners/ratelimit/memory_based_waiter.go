// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

package ratelimit

import (
	"errors"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

// MemoryBasedWaiter is a rate limiter based on memory usage. $$$$$$$$$$$$$$$$$$$$$$ TODDO
type MemoryBasedWaiter struct {
	telemetry               *MemoryBasedWaiterTelemetry
	memoryUsageRate         *memoryUsageRate
	lowSoftLimitRate        float32
	highSoftLimitRate       float32
	memoryRateLimiter       *geometricRateLimiter
	freeOSMemoryRateLimiter *geometricRateLimiter
	previousMemoryUsageRate float32
}

var memoryBasedWaiterTelemetry = &MemoryBasedWaiterTelemetry{}

func BuildMemoryBasedWaiter() (*MemoryBasedWaiter, error) {
	memoryUsageRate, err := newMemoryUsageRate()
	if err != nil {
		return nil, err
	}

	return NewMemoryBasedWaiter(
		memoryBasedWaiterTelemetry,
		memoryUsageRate,
		0, // lowSoftLimit uint64,
		0, // highSoftLimit uint64,
		0, // goGC int,
		geometricRateLimiterConfig{ // memoryRateLimiter ,
			0, 0, 0},
		geometricRateLimiterConfig{0, 0, 0}, // freeOSMemoryRateLimiter RateLimiterConfig
	)
}

// NewMemoryBasedWaiter creates a new instance of MemoryBasedWaiter.
func NewMemoryBasedWaiter(
	telemetry *MemoryBasedWaiterTelemetry,
	memoryUsageRate *memoryUsageRate,
	lowSoftLimitRate float32,
	highSoftLimitRate float32,
	goGC int,
	memoryRateLimiter geometricRateLimiterConfig,
	freeOSMemoryRateLimiter geometricRateLimiterConfig) (*MemoryBasedWaiter, error) {

	if goGC > 0 {
		debug.SetGCPercent(goGC)
	}

	if !strings.Contains(os.Getenv("GODEBUG"), "madvdontneed=1") {
		return nil, errors.New("GODEBUG must have `madvdontneed=1` in order to use this feature")
	}

	return &MemoryBasedWaiter{
		telemetry:               telemetry,
		memoryUsageRate:         memoryUsageRate,
		lowSoftLimitRate:        lowSoftLimitRate,
		highSoftLimitRate:       highSoftLimitRate,
		memoryRateLimiter:       newGeometricRateLimiter(memoryRateLimiter),
		freeOSMemoryRateLimiter: newGeometricRateLimiter(freeOSMemoryRateLimiter),
	}, nil
}

// Wait TODO
func (m *MemoryBasedWaiter) Wait() error {
	if !m.memoryRateLimiter.limitExceeded() {
		m.telemetry.IncNoWait()
		return nil
	}
	m.telemetry.IncWait()

	rate, err := m.memoryUsageRate.rate()
	if err != nil {
		return err
	}

	if rate > m.previousMemoryUsageRate {
		m.memoryRateLimiter.increaseRate()
	} else {
		m.memoryRateLimiter.decreaseRate()
	}

	for rate > m.highSoftLimitRate {
		m.telemetry.IncHighLimit()
		debug.FreeOSMemory()
		time.Sleep(100 * time.Millisecond)

		if rate, err = m.memoryUsageRate.rate(); err != nil {
			return err
		}
	}
	if rate > m.lowSoftLimitRate {
		m.telemetry.IncLowLimit()
		if m.freeOSMemoryRateLimiter.limitExceeded() {
			time.Sleep(1 * time.Millisecond)
		} else {
			debug.FreeOSMemory()
		}

		if rate > m.previousMemoryUsageRate {
			m.freeOSMemoryRateLimiter.increaseRate()
		} else {
			m.freeOSMemoryRateLimiter.decreaseRate()
		}
	}
	m.previousMemoryUsageRate = rate
	return nil
}
