// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package cgroups

import (
	"strconv"
	"time"
)

func (c *cgroupV1) GetCPUStats(stats *CPUStats) error {
	if stats == nil {
		return &InvalidInputError{Desc: "input stats cannot be nil"}
	}

	if !c.controllerMounted("cpuacct") {
		return &ControllerNotFoundError{Controller: "cpuacct"}
	}
	if !c.controllerMounted("cpu") {
		return &ControllerNotFoundError{Controller: "cpu"}
	}

	c.parseCPUAcctController(stats)
	c.parseCPUController(stats)

	// Do not raise error if `cpuset` is not present as it's not used to retrieve key features
	if c.controllerMounted("cpuset") {
		c.parseCPUSetController(stats)
	}

	return nil
}

func (c *cgroupV1) parseCPUController(stats *CPUStats) {
	if err := parseSingleUnsignedStat(c.fr, c.pathFor("cpu", "cpu.shares"), &stats.Shares); err != nil {
		reportError(err)
	}

	if err := parse2ColumnStatsWithMapping(c.fr, c.pathFor("cpu", "cpu.stat"), 0, 1, map[string]**uint64{
		"nr_throttled":   &stats.ThrottledPeriods,
		"throttled_time": &stats.ThrottledTime,
		"nr_periods":     &stats.ElapsedPeriods,
	}); err != nil {
		reportError(err)
	}

	if err := parseSingleUnsignedStat(c.fr, c.pathFor("cpu", "cpu.cfs_period_us"), &stats.SchedulerPeriod); err == nil {
		stats.SchedulerPeriod = uint64Ptr(*stats.SchedulerPeriod * uint64(time.Microsecond))
	} else {
		reportError(err)
	}

	var tempValue *int64
	if err := parseSingleSignedStat(c.fr, c.pathFor("cpu", "cpu.cfs_quota_us"), &tempValue); err == nil {
		if *tempValue != -1 {
			stats.SchedulerQuota = uint64Ptr(uint64(*tempValue) * uint64(time.Microsecond))
		}
	} else {
		reportError(err)
	}
}

func (c *cgroupV1) parseCPUAcctController(stats *CPUStats) {
	if err := parseSingleUnsignedStat(c.fr, c.pathFor("cpuacct", "cpuacct.usage"), &stats.Total); err != nil {
		reportError(err)
	}

	if err := parse2ColumnStats(c.fr, c.pathFor("cpuacct", "cpuacct.stat"), 0, 1, parseV1CPUAcctStatFn(stats)); err != nil {
		reportError(err)
	}
}

func (c *cgroupV1) parseCPUSetController(stats *CPUStats) {
	// Normally there's only one line, but as the parser works line by line anyway, we do support multiple lines
	var cpuCount uint64
	err := parseFile(c.fr, c.pathFor("cpuset", "cpuset.cpus"), func(line string) error {
		cpuCount += ParseCPUSetFormat(line)
		return nil
	})

	if err != nil {
		reportError(err)
	} else if cpuCount != 0 {
		stats.CPUCount = &cpuCount
	}
}

func parseV1CPUAcctStatFn(stats *CPUStats) func(key, val string) error {
	return func(key, val string) error {
		intVal, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			reportError(newValueError(val, err))
			return nil
		}

		switch key {
		case "user":
			stats.User = uint64Ptr(intVal * UserHZToNano)
		case "system":
			stats.System = uint64Ptr(intVal * UserHZToNano)
		}

		return nil
	}
}
