// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

package ratelimit

import (
	"os"

	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/gopsutil/mem"
	"github.com/DataDog/gopsutil/process"
)

var _ memoryUsage = (*processMemoryUsage)(nil)

type processMemoryUsage struct {
	process                   *process.Process
	optionalCgroupMemoryLimit *cgroupMemory
	totalMemory               uint64
}

func newProcessMemoryUsage() (*processMemoryUsage, error) {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return nil, err
	}

	memoryStats, err := mem.VirtualMemory()
	if err != nil || memoryStats.Total == 0 {
		return nil, err
	}

	var cgroupMemoryLimit *cgroupMemory = nil
	if cgroupMemoryLimit, err = newCgroupMemory(); err != nil {
		log.Info("No cgroup memory limit found")
		cgroupMemoryLimit = nil
	}
	return &processMemoryUsage{
		process:                   p,
		optionalCgroupMemoryLimit: cgroupMemoryLimit,
		totalMemory:               memoryStats.Total,
	}, nil
}

func (m *processMemoryUsage) rate() (float64, error) {
	memory, err := m.process.MemoryInfo()
	if err != nil {
		return 0, err
	}
	var memoryLimit uint64

	if m.optionalCgroupMemoryLimit != nil {
		if memoryLimit, err = m.optionalCgroupMemoryLimit.getMemoryLimit(); err != nil {
			return 0, err
		}
	} else {
		memoryLimit = m.totalMemory
	}

	return float64(memory.RSS) / float64(memoryLimit), nil
}
