// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.
//go:build linux
// +build linux

package ratelimit

import (
	"errors"
	"time"

	"github.com/DataDog/datadog-agent/pkg/util/cgroups"
)

var cgroupNotSupportedError = errors.New("Not supported")

type cgroupMemoryLimit struct {
	reader *cgroups.Reader
}

func newCgroupMemoryLimit() (*cgroupMemoryLimit, error) {
	reader, err := cgroups.NewReader()
	if err != nil {
		return nil, err
	}
	return &cgroupMemoryLimit{
		reader: reader,
	}, nil
}

func (c *cgroupMemoryLimit) getMemoryLimits() (uint64, error) {
	c.reader.RefreshCgroups(15 * time.Minute)
	cgroups := c.reader.ListCgroups()
	if len(cgroups) == 0 {
		return 0, nil
	}
	var stats cgroups.MemoryStats
	if err := cgroups[0].GetMemoryStats(&stats); err != nil {
		return 0, err
	}
	if stats.Limit != nil || *stats.Limit == 0 {
		return 0, errors.New("Cannot get memory limit")
	}
	return stats.Limit, nil
}
