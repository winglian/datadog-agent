//  Unless explicitly stated otherwise all files in this repository are licensed
//  under the Apache License Version 2.0.
//  This product includes software developed at Datadog (https://www.datadoghq.com/).
//  Copyright 2016-present Datadog, Inc.

package ebpftest

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/DataDog/datadog-agent/pkg/ebpf"
	"github.com/DataDog/datadog-agent/pkg/process/util"
	"github.com/DataDog/datadog-agent/pkg/security/utils"
)

type tracePipeLogger struct {
	*TracePipe
	t          *testing.T
	stop       chan struct{}
	executable string
}

func (l *tracePipeLogger) handleEvent(event *TraceEvent) {
	// for some reason, the event task is resolved to "<...>"
	// so we check that event.PID is the ID of a task of the running process
	taskPath := filepath.Join(util.HostProc(), strconv.Itoa(int(utils.Getpid())), "task", event.PID)
	_, err := os.Stat(taskPath)

	if event.Task == l.executable || (event.Task == "<...>" && err == nil) {
		l.t.Log(event.Raw)
	}
}

func (l *tracePipeLogger) Start() {
	channelEvents, channelErrors := l.Channel()

	go func() {
		for {
			select {
			case <-l.stop:
				for len(channelEvents) > 0 {
					l.handleEvent(<-channelEvents)
				}
				return
			case event := <-channelEvents:
				l.handleEvent(event)
			case err := <-channelErrors:
				l.t.Logf("trace_pipe error: %s", err)
			}
		}
	}()
}

func (l *tracePipeLogger) Stop() {
	time.Sleep(time.Millisecond * 200)

	l.stop <- struct{}{}
	_ = l.Close()
}

// StartTracing starts capturing the output of the kprobe trace_pipe for the current running process
func StartTracing(t *testing.T, cfg *ebpf.Config) {
	// force this to on, because tracer is worthless otherwise
	cfg.BPFDebug = true
	tracePipe, err := NewTracePipe()
	if err != nil {
		t.Error(err)
	}

	executable, err := os.Executable()
	if err != nil {
		t.Error(err)
	}

	logger := &tracePipeLogger{
		t:          t,
		TracePipe:  tracePipe,
		stop:       make(chan struct{}),
		executable: filepath.Base(executable),
	}
	t.Cleanup(logger.Stop)
	logger.Start()

	time.Sleep(time.Millisecond * 200)
}
