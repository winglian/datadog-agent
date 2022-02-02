// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

package ratelimit

type geometricRateLimiterConfig struct {
	minRate float64
	maxRate float64
	factor  float64
}

type geometricRateLimiter struct {
	tick             int
	currentRateLimit float64
	minRate          float64
	maxRate          float64
	factor           float64
}

func newGeometricRateLimiter(config geometricRateLimiterConfig) *geometricRateLimiter {
	return &geometricRateLimiter{
		minRate:          config.minRate,
		maxRate:          config.maxRate,
		factor:           config.factor,
		currentRateLimit: config.minRate,
	}
}

func (r *geometricRateLimiter) limitExceeded() bool {
	r.tick++
	if 1/float64(r.tick) <= r.currentRateLimit {
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
