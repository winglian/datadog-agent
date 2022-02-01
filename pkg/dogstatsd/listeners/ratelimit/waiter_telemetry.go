// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

package ratelimit

import (
	"github.com/DataDog/datadog-agent/pkg/telemetry"
)

type MemoryBasedWaiterTelemetry struct {
	wait      telemetry.Counter
	noWait    telemetry.Counter
	highLimit telemetry.Counter
	lowLimit  telemetry.Counter
}

func NewMemoryBasedWaiterTelemetry() *MemoryBasedWaiterTelemetry {
	return &MemoryBasedWaiterTelemetry{
		wait:      telemetry.NewCounter("dogstatsd", "memory_based_waiter_wait", []string{}, "The number of times Wait() blocks"),
		noWait:    telemetry.NewCounter("dogstatsd", "memory_based_waiter_no_wait", []string{}, "The number of times Wait() doesn't blocks"),
		highLimit: telemetry.NewCounter("dogstatsd", "memory_based_waiter_high_limit", []string{}, "The number of times high limit is reached"),
		lowLimit:  telemetry.NewCounter("dogstatsd", "memory_based_waiter_low_limit", []string{}, "The number of times soft limit is reached"),
	}
}

func (t *MemoryBasedWaiterTelemetry) IncWait() {
	t.wait.Inc()
}
func (t *MemoryBasedWaiterTelemetry) IncNoWait() {
	t.noWait.Inc()
}
func (t *MemoryBasedWaiterTelemetry) IncHighLimit() {
	t.highLimit.Inc()
}
func (t *MemoryBasedWaiterTelemetry) IncLowLimit() {
	t.lowLimit.Inc()
}
