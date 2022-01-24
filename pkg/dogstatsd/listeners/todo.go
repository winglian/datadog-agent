// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package listeners

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/gopsutil/process"
)

type Todo struct {
	tick  int
	count int
	alert int
}

func NewTodo() *Todo {

	size, err := strconv.Atoi(os.Getenv("MEMORY_BALLAST"))
	if err != nil {
		log.Error("Invalid value for ", os.Getenv("MEMORY_BALLAST"))
	} else {
		log.Error("Ballast size", size)
		ballast := make([]byte, size)
		runtime.KeepAlive(ballast)
	}
	return &Todo{}
}

func (t *Todo) OnNewPacket() {
	t.tick++
	t.count++
	if true { // t.tick > 10 {
		p, err := process.NewProcess(int32(os.Getpid()))
		if err != nil {
			log.Error("", err)
		}
		stats, err := p.MemoryInfo()
		if err != nil {
			log.Error("", err)
		}

		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		//		log.Info(t.count, " ", stats.RSS/(1024*1024), m.HeapInuse, m.HeapSys)

		for stats.RSS > uint64(config.Datadog.GetInt("memomry_limit2")) {
			runtime.ReadMemStats(&m)
			log.Info("mem", t.count, " ", stats.RSS/(1024*1024))
			//log.Info("mem", t.count, " ", stats.RSS/(1024*1024), " ", get_string(&m))
			//if config.Datadog.GetBool("memomry_limit_run_gc") {
			t.alert++
			//	if t.alert%10 == 0 {
			//	runtime.GC()
			debug.FreeOSMemory()
			//	}
			//}

			if os.Getenv("GODEBUG") != "madvdontneed=1" {
				panic("GODEBUG != madvdontneed=1")
			}

			time.Sleep(time.Duration(config.Datadog.GetInt("memomry_limit1_wait_duration_ms")) * time.Millisecond)
			p, err = process.NewProcess(int32(os.Getpid()))
			if err != nil {
				log.Error("", err)
			}
			stats, err = p.MemoryInfo()
			if err != nil {
				log.Error("", err)
			}
		}
		if stats.RSS > uint64(config.Datadog.GetInt("memomry_limit1")) {
			diff := stats.RSS / (1024 * 1024)
			var d time.Duration
			if diff < 550 {
				d = time.Millisecond
			} else if diff < 600 {
				d = 100 * time.Millisecond
			} else {
				d = 1000 * time.Millisecond
			}

			time.Sleep(time.Duration(d))
			runtime.ReadMemStats(&m)
			log.Info("mem2", t.count, " ", stats.RSS/(1024*1024))
			//		log.Info("mem2", t.count, " ", stats.RSS/(1024*1024), " ", get_string(&m))
			//if t.tick%20 == 0 {
			t.alert++
			//	if t.alert%10 == 0 {
			//		runtime.GC()
			debug.FreeOSMemory()
			//	}
			//}
		} else {
			//		log.Info("OK")
			t.tick = 0
		}
		//	log.Info("READING")
	}
}

func get_string(m *runtime.MemStats) string {
	return fmt.Sprintln(
		"Alloc", m.Alloc,
		"TotalAlloc", m.TotalAlloc,
		"Sys", m.Sys,
		"Mallocs", m.Mallocs,
		"Frees", m.Frees,
		"HeapAlloc", m.HeapAlloc,
		"HeapSys", m.HeapSys,
		"HeapIdle", m.HeapIdle,
		"HeapInuse", m.HeapInuse,
		"HeapReleased", m.HeapReleased,
		"HeapObjects", m.HeapObjects,
		"StackInuse", m.StackInuse,
		"StackSys", m.StackSys,
		"MSpanInuse", m.MSpanInuse,
		"MSpanSys", m.MSpanSys,
		"MCacheInuse", m.MCacheInuse,
		"MCacheSys", m.MCacheSys,
		"BuckHashSys", m.BuckHashSys,
		"GCSys", m.GCSys,
		"OtherSys", m.OtherSys,
	)

}

// 2022-01-24 11:59:26 CET | CORE | INFO | (pkg/dogstatsd/listeners/todo.go:86 in OnNewPacket) | mem2 272007   586
// 2022-01-24 11:59:31 CET | CORE | INFO | (pkg/dogstatsd/listeners/todo.go:49 in OnNewPacket) | mem 272008   1105

// GODEBUG=madvdontneed=1
// /bin/agent/agent run -c ./bin/agent/dist/ 2> /dev/null | grep "todo.go"
// 0 -> 350
// 2  350
// 3  350
// 4  200

// RSS increase whereas heapSys and heapInUse are stable
// 022-01-20 19:48:46 CET | CORE | INFO | (pkg/dogstatsd/listeners/todo.go:75 in OnNewPacket) | Wait:  1s 624
// 2022-01-20 19:48:47 CET | CORE | INFO | (pkg/dogstatsd/listeners/todo.go:84 in OnNewPacket) | READING
// 2022-01-20 19:48:47 CET | CORE | INFO | (pkg/dogstatsd/listeners/todo.go:44 in OnNewPacket) | 51907   709 621912064 1005682688
// 2022-01-20 19:48:47 CET | CORE | INFO | (pkg/dogstatsd/listeners/todo.go:55 in OnNewPacket) | 4242-2  51907   709   365  WAIT 621912064  SYS: 1005682688
// 2022-01-20 19:48:49 CET | CORE | INFO | (pkg/dogstatsd/listeners/todo.go:55 in OnNewPacket) | 4242-2  51907   948   365  WAIT 621912064  SYS: 1005682688
// 2022-01-20 19:48:52 CET | CORE | INFO | (pkg/dogstatsd/listeners/todo.go:55 in OnNewPacket) | 4242-2  51907   917   365  WAIT 621912064  SYS: 1005682688

// mem 37851   1101   Alloc 759744096 TotalAlloc 2412567312 Sys 1135328691 Mallocs 23077119 Frees 12359229 HeapAlloc 759744096 HeapSys 1072726016 HeapIdle 283754496 HeapInuse 788971520 HeapReleased 42909696 HeapObjects 10717890 StackInuse 1015808 StackSys 1015808 MSpanInuse 7804904 MSpanSys 11304960 MCacheInuse 7200 MCacheSys 16384 BuckHashSys 1607060 GCSys 46401160 OtherSys 2257303
// mem 37851   476    Alloc 759744096 TotalAlloc 2412567312 Sys 1135328691 Mallocs 23077119 Frees 12359229 HeapAlloc 759744096 HeapSys 1072726016 HeapIdle 283754496 HeapInuse 788971520 HeapReleased 42909696 HeapObjects 10717890 StackInuse 1015808 StackSys 1015808 MSpanInuse 7804904 MSpanSys 11304960 MCacheInuse 7200 MCacheSys 16384 BuckHashSys 1607060 GCSys 46401160 OtherSys 2257303

// 2022-01-20 20:10:06 CET | CORE | INFO | (pkg/dogstatsd/listeners/todo.go:77 in OnNewPacket) | mem2 40648   606   Alloc 861064024 TotalAlloc 3211521664 Sys 921030019 Mallocs 29027150 Frees 19895803 HeapAlloc 861064024 HeapSys 871432192 HeapIdle 4857856   HeapInuse 866574336 HeapReleased 3653632   HeapObjects 9131347  StackInuse 983040 StackSys 983040 MSpanInuse 8246496 MSpanSys 8290304 MCacheInuse 7200 MCacheSys 16384 BuckHashSys 1596124 GCSys 36935248 OtherSys 1776727
// 2022-01-20 20:10:07 CET | CORE | INFO | (pkg/dogstatsd/listeners/todo.go:56 in OnNewPacket) | mem 40649   711    Alloc 620061072 TotalAlloc 3292884648 Sys 990566803 Mallocs 29910216 Frees 21534786 HeapAlloc 620061072 HeapSys 938541056 HeapIdle 309927936 HeapInuse 628613120 HeapReleased 309051392 HeapObjects 8375430  StackInuse 983040 StackSys 983040 MSpanInuse 5275576 MSpanSys 8421376 MCacheInuse 7200 MCacheSys 16384 BuckHashSys 1596124 GCSys 39105144 OtherSys 1903679
// 2022-01-20 20:10:09 CET | CORE | INFO | (pkg/dogstatsd/listeners/todo.go:56 in OnNewPacket) | mem 40649   832    Alloc 747951784 TotalAlloc 3565470248 Sys 990763411 Mallocs 34209736 Frees 22838226 HeapAlloc 747951784 HeapSys 938541056 HeapIdle 182648832 HeapInuse 755892224 HeapReleased 182067200 HeapObjects 11371510 StackInuse 983040 StackSys 983040 MSpanInuse 7531136 MSpanSys 8421376 MCacheInuse 7200 MCacheSys 16384 BuckHashSys 1596932 GCSys 39301752 OtherSys 1902871
