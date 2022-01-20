// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package listeners

import (
	"os"
	"runtime"
	"time"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/gopsutil/process"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

type Todo struct {
	tick  int
	count int
}

func NewTodo() *Todo {
	return &Todo{}
}

func (t *Todo) OnNewPacket() {
	t.tick++
	t.count++
	if t.tick > 100 {
		p, err := process.NewProcess(int32(os.Getpid()))
		if err != nil {
			log.Error("", err)
		}
		stats, err := p.MemoryInfo()
		if err != nil {
			log.Error("", err)
		}

		// var m runtime.MemStats
		// runtime.ReadMemStats(&m)
		log.Info(t.count, " ", stats.RSS/(1024*1024))

		for stats.RSS > uint64(config.Datadog.GetInt("memomry_limit2")) {
			log.Info("4242 ", t.count, " ", stats.RSS/(1024*1024), " WAIT")
			if config.Datadog.GetBool("memomry_limit_run_gc") {
				runtime.GC()
			}

			time.Sleep(time.Duration(config.Datadog.GetInt("memomry_limit1_wait_duration_ms")) * time.Millisecond)
		}
		if stats.RSS > uint64(config.Datadog.GetInt("memomry_limit1")) {
			log.Info("4242 Wait")
			time.Sleep(1 * time.Second)
		}
		t.tick = 0
	}
}

// 0 -> 350
// 2  350
// 3  350
// 4  200
