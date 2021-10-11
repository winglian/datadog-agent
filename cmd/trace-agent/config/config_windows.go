// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package config

import (
	"path/filepath"

	"github.com/DataDog/datadog-agent/pkg/trace/config"
	"github.com/DataDog/datadog-agent/pkg/util/executable"
)

func init() {
	pd, err := winutil.GetProgramDataDir()
	if err == nil {
		config.DefaultLogFilePath = filepath.Join(pd, "logs", "trace-agent.log")
	}
	_here, err := executable.Folder()
	if err == nil {
		DefaultDDAgentBin = filepath.Join(_here, "..", "agent.exe")
	}
}
