package checks

import (
	model "github.com/DataDog/agent-payload/process"
	"github.com/DataDog/datadog-agent/pkg/process/config"
	"github.com/DataDog/datadog-agent/pkg/process/procutil"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"strings"
)

var ProcessEvents = &ProcessEventsCheck{probe: procutil.NewProcessProbe()}

type ProcessEventsCheck struct {
	probe      *procutil.Probe
	info       *model.SystemInfo
}

func (c *ProcessEventsCheck) Init(_ *config.AgentConfig, info *model.SystemInfo) {
}

func (c *ProcessEventsCheck) Name() string { return config.ProcessEventsCheckName }

func (c *ProcessEventsCheck) RealTime() bool { return false }


func (c *ProcessEventsCheck) Run(cfg *config.AgentConfig, groupID int32) ([]model.MessageBody, error) {
	log.Info("Running process_events check")

	procs, err := c.probe.GetCreatedProcesses()
	if err != nil {
		log.Error("Error fetching CreatedProcess")
	}
	for _, proc := range procs {
		log.Infof("=== process created {PID: %d CMD: %s USER: %s CREATE_TIME: %d }",
			proc.Pid, strings.Join(proc.Cmdline, " "), proc.Username, proc.Stats.CreateTime)
	}

	return nil, nil
}


