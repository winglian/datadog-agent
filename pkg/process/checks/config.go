// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package checks

import (
	ddconfig "github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

func getBatchSize() int {
	batchSize := ddconfig.Datadog.GetInt("process_config.max_per_message")
	if batchSize <= 0 {
		log.Warnf("Invalid item count per message (<= 0), using default value of %d", ddconfig.DefaultProcessMaxPerMessage)
		batchSize = ddconfig.DefaultProcessMaxPerMessage
	} else if batchSize > ddconfig.DefaultProcessMaxPerMessage {
		log.Warnf("Overriding the configured max of item count per message because it exceeds maximum limit of %d", ddconfig.DefaultProcessMaxPerMessage)
		batchSize = ddconfig.DefaultProcessMaxPerMessage
	}

	return batchSize
}

func getCtrProcsBatchSize() int {
	ctrProcsBatchSize := ddconfig.Datadog.GetInt("process_config.max_ctr_procs_per_message")
	if ctrProcsBatchSize <= 0 {
		log.Warnf("Invalid max container processes count per message (<= 0), using default value of %d", ddconfig.DefaultProcessMaxCtrProcsPerMessage)
		ctrProcsBatchSize = ddconfig.DefaultProcessMaxCtrProcsPerMessage
	} else if ctrProcsBatchSize > ddconfig.ProcessMaxCtrProcsPerMessageLimit {
		log.Warnf("Overriding the configured max of container processes count per message because it exceeds maximum limit of %d", ddconfig.ProcessMaxCtrProcsPerMessageLimit)
		ctrProcsBatchSize = ddconfig.DefaultProcessMaxCtrProcsPerMessage
	}
	return ctrProcsBatchSize
}

