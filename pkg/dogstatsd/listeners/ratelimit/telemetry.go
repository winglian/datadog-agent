// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

package ratelimit

import (
	tlm "github.com/DataDog/datadog-agent/pkg/telemetry"
)

type telemetry struct {
	wait              tlm.Counter
	noWait            tlm.Counter
	highLimit         tlm.Counter
	lowLimit          tlm.Counter
	lowLimitFreeOSMem tlm.Counter
	memoryUsageRate   tlm.Gauge
}

func newTelemetry() *telemetry {
	return &telemetry{
		wait:              tlm.NewCounter("dogstatsd", "mem_based_rate_limiter_wait", []string{}, "The number of times Wait() wait"),
		noWait:            tlm.NewCounter("dogstatsd", "mem_based_rate_limiter_no_wait", []string{}, "The number of times Wait() doesn't wait"),
		highLimit:         tlm.NewCounter("dogstatsd", "mem_based_rate_limiter_high_limit", []string{}, "The number of times high limit is reached"),
		lowLimit:          tlm.NewCounter("dogstatsd", "mem_based_rate_limiter_low_limit", []string{}, "The number of times soft limit is reached"),
		lowLimitFreeOSMem: tlm.NewCounter("dogstatsd", "mem_based_rate_limiter_low_limit_freeos_mem", []string{}, "The number of times FreeOSMemory is called when soft limit is reached"),
		memoryUsageRate:   tlm.NewGauge("dogstatsd", "mem_based_rate_limiter_mem_rate", []string{}, "The memory usage rate"),
	}
}

func (t *telemetry) incWait() {
	t.wait.Inc()
}

func (t *telemetry) incNoWait() {
	t.noWait.Inc()
}

func (t *telemetry) incHighLimit() {
	t.highLimit.Inc()
}

func (t *telemetry) incLowLimit() {
	t.lowLimit.Inc()
}

func (t *telemetry) incLowLimitFreeOSMemory() {
	t.lowLimitFreeOSMem.Inc()
}

func (t *telemetry) setMemoryUsageRate(rate float64) {
	t.memoryUsageRate.Set(rate)
}
