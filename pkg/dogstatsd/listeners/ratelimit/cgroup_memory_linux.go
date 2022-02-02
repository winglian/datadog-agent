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

type cgroupMemory struct {
	reader *cgroups.Reader
}

func newCgroupMemory() (*cgroupMemory, error) {
	reader, err := cgroups.NewReader()
	if err != nil {
		return nil, err
	}
	limit := &cgroupMemory{
		reader: reader,
	}

	// make sure a memory limit is defined
	if _, err := limit.getMemoryLimit(); err != nil {
		return nil, err
	}

	return limit, nil
}

func (c *cgroupMemory) getMemoryLimit() (uint64, error) {
	c.reader.RefreshCgroups(15 * time.Minute)
	groups := c.reader.ListCgroups()
	if len(groups) == 0 {
		return 0, nil
	}
	var stats cgroups.MemoryStats
	if err := groups[0].GetMemoryStats(&stats); err != nil {
		return 0, err
	}
	if stats.Limit == nil || *stats.Limit == 0 {
		return 0, errors.New("cannot get memory limit")
	}
	return *stats.Limit, nil
}
