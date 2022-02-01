// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

package ratelimit

import (
	"os"

	"github.com/DataDog/gopsutil/mem"
	"github.com/DataDog/gopsutil/process"
)

type memoryUsageRate struct {
	process                   *process.Process
	optionalCgroupMemoryLimit *cgroupMemoryLimit
	totalMemory               uint64
}

func newMemoryUsageRate() (*memoryUsageRate, error) {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return nil, err
	}

	memoryStats, err := mem.VirtualMemory()
	if err != nil || memoryStats.Total == 0 {
		return nil, err
	}

	var cgroupMemoryLimit *cgroupMemoryLimit = nil
	if cgroupMemoryLimit, err = newCgroupMemoryLimit(); err != nil {
		if err != cgroupNotSupportedError {
			return nil, err
		}
		cgroupMemoryLimit = nil
	}
	return &memoryUsageRate{
		process:                   p,
		optionalCgroupMemoryLimit: cgroupMemoryLimit,
		totalMemory:               memoryStats.Total,
	}, nil
}

func (m *memoryUsageRate) rate() (float32, error) {
	memory, err := m.process.MemoryInfo()
	if err != nil {
		return 0, err
	}
	var memoryLimit uint64

	if m.optionalCgroupMemoryLimit != nil {
		if memoryLimit, err = m.optionalCgroupMemoryLimit.getMemoryLimits(); err != nil {
			return 0, nil
		}
	} else {
		memoryLimit = m.totalMemory
	}

	return float32(memory.RSS) / float32(memoryLimit), nil
}
