// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

package listeners

import (
	"errors"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/telemetry"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/gopsutil/process"
)

var tlmIn = telemetry.NewCounter("memory_soft_limit", "in", []string{}, "")
var tlmOut = telemetry.NewCounter("memory_soft_limit", "out", []string{}, "")
var tlmMemory = telemetry.NewGauge("memory_soft_limit", "memory", []string{}, "")
var tlmHigh = telemetry.NewCounter("memory_soft_limit", "high", []string{}, "")
var tlmLow = telemetry.NewCounter("memory_soft_limit", "low", []string{}, "")
var tlmFreeOSMemory = telemetry.NewCounter("memory_soft_limit", "free_os_memory", []string{}, "")

var tlmRate = telemetry.NewGauge("memory_soft_limit", "rate", []string{}, "")
var tlmRateFreeOs = telemetry.NewGauge("memory_soft_limit", "rate_free_os", []string{}, "")

var ballast sync.Once

// MemThrottle TODO
type MemThrottle struct {
	process                *process.Process
	soft_memory_limit_low  uint64
	soft_memory_limit_high uint64
	main                   *RateLimiter
	os_release             *RateLimiter
	previous_rss           uint64
}

// NewMemThrottle TODO
func NewMemThrottle() (*MemThrottle, error) {
	ballastSize := uint64(config.Datadog.GetInt("soft_memory_limit_ballast"))
	low := uint64(config.Datadog.GetInt("soft_memory_limit_low"))
	high := uint64(config.Datadog.GetInt("soft_memory_limit_high"))
	goGc := config.Datadog.GetInt("soft_memory_limit_go_gc")
	log.Infof("low:%v high:%v ballastSize:%v gogc:%v", low, high, ballastSize, goGc)

	if goGc > 0 {
		debug.SetGCPercent(goGc)
	}
	if ballastSize > 0 {
		ballast.Do(func() {
			ballast := make([]byte, ballastSize)
			runtime.KeepAlive(ballast)
		})
	}

	if !strings.Contains(os.Getenv("GODEBUG"), "madvdontneed=1") {
		return nil, errors.New("GODEBUG must have `madvdontneed=1` in order to use this feature")
	}

	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return nil, err
	}

	rate_min := config.Datadog.GetFloat64("soft_memory_limit_rate_min")
	rate_max := config.Datadog.GetFloat64("soft_memory_limit_rate_max")
	rate_factor := config.Datadog.GetFloat64("soft_memory_limit_rate_factor")
	release_rate_min := config.Datadog.GetFloat64("soft_memory_limit_release_rate_min")
	release_rate_max := config.Datadog.GetFloat64("soft_memory_limit_release_rate_max")
	release_rate_factor := config.Datadog.GetFloat64("soft_memory_limit_release_rate_factor")
	log.Infof("rate_min:%v rate_max:%v rate_factor:%v release_rate_min:%v release_rate_max:%v release_rate_factor:%v", rate_min, rate_max, rate_factor, release_rate_min, release_rate_max, release_rate_factor)

	return &MemThrottle{
		process:                p,
		soft_memory_limit_low:  low,
		soft_memory_limit_high: high,
		main:                   NewRateLimiter(rate_min, rate_max, rate_factor),
		os_release:             NewRateLimiter(release_rate_min, release_rate_max, release_rate_factor),
	}, nil
}

// ThrottleIfLimitReached TODO
func (t *MemThrottle) ThrottleIfLimitReached() error {
	if !t.main.Keep() {
		tlmOut.Inc()
		return nil
	}
	tlmIn.Inc()
	stats, err := t.process.MemoryInfo()
	if err != nil {
		return err
	}

	tlmMemory.Set(float64(stats.RSS))

	for stats.RSS > t.soft_memory_limit_high {
		tlmHigh.Inc()
		debug.FreeOSMemory()
		time.Sleep(100 * time.Millisecond)

		if stats, err = t.process.MemoryInfo(); err != nil {
			return err
		}
	}
	if stats.RSS > t.soft_memory_limit_low {
		tlmLow.Inc()
		if t.os_release.Keep() {
			debug.FreeOSMemory()
			tlmFreeOSMemory.Inc()
		} else {
			time.Sleep(1 * time.Millisecond)
		}

		if stats.RSS > t.previous_rss {
			t.main.Increase()
			t.os_release.Increase()
		} else {
			t.os_release.Decrease()
		}
		t.previous_rss = stats.RSS
	} else {
		t.main.Decrease()
	}
	tlmRate.Set(t.main.Rate())
	tlmRateFreeOs.Set(t.os_release.Rate())
	return nil
}

// RateLimiter TODO
type RateLimiter struct {
	tick     int
	value    float64 // $$ rename to limit?
	minValue float64
	maxValue float64
	factor   float64
}

// NewRateLimiter TODO
func NewRateLimiter(minValue float64, maxValue float64, factor float64) *RateLimiter {
	// minValue should be at least 1 TODO
	return &RateLimiter{
		minValue: minValue,
		maxValue: maxValue,
		factor:   factor,
		value:    maxValue,
	}
}

// Rate TODO
func (r *RateLimiter) Rate() float64 {
	return 1 / r.value
}

// Keep TODO
func (r *RateLimiter) Keep() bool {
	r.tick++
	if float64(r.tick) >= r.value {
		r.tick = 0
		return true
	}
	return false
}

// Increase TODO
func (r *RateLimiter) Increase() {
	r.value /= r.factor
	r.normalize()
}

// Decrease TODO
func (t *RateLimiter) Decrease() {
	t.value *= t.factor
	t.normalize()
}

func (t *RateLimiter) normalize() {
	if t.value > t.maxValue {
		t.value = t.maxValue
	}
	if t.value < t.minValue {
		t.value = t.minValue
	}
}

// 2022-01-24 11:59:26 CET | CORE | INFO | (pkg/dogstatsd/listeners/todo.go:86 in OnNewPacket) | mem2 272007   586
// 2022-01-24 11:59:31 CET | CORE | INFO | (pkg/dogstatsd/listeners/todo.go:49 in OnNewPacket) | mem 272008   1105

// GODEBUG=madvdontneed=1
// gctrace=1
// /bin/agent/agent run -c ./bin/agent/dist/ 2> /dev/null | grep "todo.go"

// GOGC=100 -> 907MB
// GOGC=50 -> 877MB
// GOGC=50  -> 845MB
// GOGC=10 -> 700MB

// debug.FreeOSMemory() -> force GC
// Ballast help
