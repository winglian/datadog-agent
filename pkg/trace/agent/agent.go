// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package agent

import (
	"context"

	"github.com/DataDog/datadog-agent/pkg/trace/api"
	"github.com/DataDog/datadog-agent/pkg/trace/config"
	"github.com/DataDog/datadog-agent/pkg/trace/pipeline"
	"github.com/DataDog/datadog-agent/pkg/trace/sampler"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// Agent struct holds all the sub-routines structs and make the data flow between them
type Agent struct {
	Processor    *pipeline.Processor
	Receiver     *api.HTTPReceiver
	OTLPReceiver *api.OTLPReceiver

	// Used to synchronize on a clean exit
	ctx context.Context
}

// NewAgent returns a new Agent object, ready to be started. It takes a context
// which may be cancelled in order to gracefully stop the agent.
func NewAgent(ctx context.Context, conf *config.AgentConfig) *Agent {
	dynConf := sampler.NewDynamicConfig(conf.DefaultEnv)
	var httprcv *api.HTTPReceiver
	proc := pipeline.NewProcessor(&pipeline.Config{
		Agent: conf,
		Rates: dynConf,
		PreSampleRate: func() (float64, bool) {
			if httprcv == nil {
				return 1, false
			}
			return httprcv.RateLimiter.RealRate(), httprcv.RateLimiter.Active()
		},
	})
	httprcv = api.NewHTTPReceiver(conf, dynConf, proc.In, proc)
	agnt := &Agent{
		ctx:          ctx,
		Processor:    proc,
		Receiver:     httprcv,
		OTLPReceiver: api.NewOTLPReceiver(proc.In, conf.OTLPReceiver),
	}
	return agnt
}

// Run starts routers routines and individual pieces then stop them when the exit order is received
func (a *Agent) Run() {
	for _, starter := range []interface{ Start() }{
		a.Receiver,
		a.OTLPReceiver,
		a.Processor,
	} {
		starter.Start()
	}
	for {
		select {
		case <-a.ctx.Done():
			log.Info("Exiting...")
			if err := a.Receiver.Stop(); err != nil {
				log.Error(err)
			}
			a.Processor.Stop()
			return
		}
	}
}

// FlushSync flushes traces sychronously. This method only works when the agent is configured in synchronous flushing
// mode via the apm_config.sync_flush option.
func (a *Agent) FlushSync() {
	if err := a.Processor.FlushSync(); err != nil {
		log.Errorf("Error flushing synchronously stats: %s", err.Error())
	}
}

// SetGlobalTagsUnsafe sets global tags to the agent configuration. Unsafe for concurrent use.
func (a *Agent) SetGlobalTagsUnsafe(tags map[string]string) {
	a.Processor.SetGlobalTagsUnsafe(tags)
}
