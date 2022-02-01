// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

package ratelimit

type geometricRateLimiterConfig struct {
	minValue float32
	maxValue float32
	factor   float32
}

type geometricRateLimiter struct {
	tick             int
	currentRateLimit float32
	minRate          float32
	maxRate          float32
	factor           float32
}

func newGeometricRateLimiter(config geometricRateLimiterConfig) *geometricRateLimiter {
	return &geometricRateLimiter{
		minRate:          config.minValue,
		maxRate:          config.maxValue,
		factor:           config.factor,
		currentRateLimit: config.maxValue,
	}
}

func (r *geometricRateLimiter) rate() float32 {
	return r.currentRateLimit
}

func (r *geometricRateLimiter) limitExceeded() bool {
	r.tick++
	if 1/float32(r.tick) >= r.currentRateLimit {
		r.tick = 0
		return true
	}
	return false
}

func (r *geometricRateLimiter) increaseRate() {
	r.currentRateLimit *= r.factor
	r.normalizeRate()
}

func (r *geometricRateLimiter) decreaseRate() {
	r.currentRateLimit /= r.factor
	r.normalizeRate()
}

func (r *geometricRateLimiter) normalizeRate() {
	if r.currentRateLimit > r.maxRate {
		r.currentRateLimit = r.maxRate
	}
	if r.currentRateLimit < r.minRate {
		r.currentRateLimit = r.minRate
	}
}
