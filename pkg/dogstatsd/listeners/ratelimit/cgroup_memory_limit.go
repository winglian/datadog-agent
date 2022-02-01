// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

//go:build !linux
// +build !linux

package ratelimit

import (
	"errors"
)

type cgroupMemoryLimit struct{}

var cgroupNotSupportedError = errors.New("Not supported")

func newCgroupMemoryLimit() (*cgroupMemoryLimit, error) {
	return nil, cgroupNotSupportedError
}

func (c *cgroupMemoryLimit) getMemoryLimits() (uint64, error) {
	return 0, cgroupNotSupportedError
}
