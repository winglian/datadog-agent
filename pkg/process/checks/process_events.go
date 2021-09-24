package checks

import (
	model "github.com/DataDog/agent-payload/process"
	"github.com/DataDog/datadog-agent/pkg/process/config"
	"github.com/DataDog/datadog-agent/pkg/process/procutil"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

var ProcessEvents = &ProcessEventsCheck{}

type ProcessEventsCheck struct {
	probe      *procutil.Probe
	info       *model.SystemInfo
}

func (c *ProcessEventsCheck) Init(_ *config.AgentConfig, info *model.SystemInfo) {
	c.probe = procutil.NewProcessProbe(procutil.WithProcessEventListener())
	c.info = info
}

func (c *ProcessEventsCheck) Name() string { return config.ProcessEventsCheckName }

func (c *ProcessEventsCheck) RealTime() bool { return false }


func (c *ProcessEventsCheck) Run(cfg *config.AgentConfig, groupID int32) ([]model.MessageBody, error) {
	log.Info("Running process_events check")

	procsByPID, err := c.probe.GetCreatedProcesses()
	procs := make([]*model.Process, 0, len(procsByPID))
	if err != nil {
		log.Error("Error fetching CreatedProcess")
	}
	for _, fp := range procsByPID {
		proc := &model.Process{
			Pid:                    fp.Pid,
			NsPid:                  fp.NsPid,
			Command:                formatCommand(fp),
			User:                 	&model.ProcessUser{
				Name: fp.Username,
			},
			Memory:                 &model.MemoryStat{},
			Cpu:                    &model.CPUStat{
				Cpus:       []*model.SingleCPUStat{},
			},
			CreateTime:             fp.Stats.CreateTime,
			OpenFdCount:            -1,
			State:                  20,
			IoStat:                 &model.IOStat{},
			VoluntaryCtxSwitches:   uint64(0),
			InvoluntaryCtxSwitches: uint64(0),
			ContainerId:            "",
		}

		log.Infof("collected proc: %v", proc)
		procs = append(procs, proc)
		//log.Infof("=== process created {PID: %d CMD: %s USER: %s CREATE_TIME: %d }",
		//	proc.Pid, strings.Join(fp.Cmdline, " "), fp.Username, fp.Stats.CreateTime)
	}

	log.Infof("COLLECTED TOTAL PROCS: %d", len(procs))
	procsByCtr := map[string][]*model.Process{
		emptyCtrID: procs,
	}
	messages, totalProcs, totalContainers := createProcCtrMessages(procsByCtr, nil, cfg, c.info, groupID, "")
	log.Infof("collected %d messages with %d processses and %d containers", len(messages), totalProcs, totalContainers)

	return messages, nil
}


