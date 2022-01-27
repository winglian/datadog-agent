// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build !windows
// +build !windows

package windowsevent

import (
	"context"

	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// Start does not do much
func (t *Tailer) Start() error {
	log.Warn("windows event log not supported on this system")
	return t.TailerBase.Start()
}

// run does nothing
func (t *Tailer) run(ctx context.Context) {
	<-ctx.Done()
}
