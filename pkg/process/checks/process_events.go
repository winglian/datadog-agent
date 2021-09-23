package checks

import (

	model "github.com/DataDog/agent-payload/process"
	"github.com/DataDog/datadog-agent/pkg/process/config"
	"github.com/DataDog/datadog-agent/pkg/process/procutil"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

var ProcessEvents = &ProcessEventsCheck{probe: procutil.NewProcessProbe()}

type ProcessEventsCheck struct {
	probe      *procutil.Probe
	info       *model.SystemInfo
}

func (d *ProcessEventsCheck) Init(_ *config.AgentConfig, info *model.SystemInfo) {
}

func (d *ProcessEventsCheck) Name() string { return config.ProcessEventsCheckName }

func (d *ProcessEventsCheck) RealTime() bool { return false }


func (d *ProcessEventsCheck) Run(cfg *config.AgentConfig, groupID int32) ([]model.MessageBody, error) {
	log.Info("Running process_events check")
	return nil, nil
}


