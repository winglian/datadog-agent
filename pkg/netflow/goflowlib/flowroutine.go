package goflowlib

import (
	"github.com/netsampler/goflow2/format"
	"github.com/netsampler/goflow2/utils"

	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// setting reusePort to false since not expected to be useful
// more info here: https://stackoverflow.com/questions/14388706/how-do-so-reuseaddr-and-so-reuseport-differ
const reusePort = false

// FlowRoutineState is a wrapper for StateNetFlow/StateSFlow/StateNFLegacy
type FlowRoutineState struct {
	state    interface{}
	hostname string
	port     uint16
}

// Shutdown is a wrapper for StateNetFlow/StateSFlow/StateNFLegacy Shutdown method
func (s *FlowRoutineState) Shutdown() {
	switch state := s.state.(type) {
	case *utils.StateNetFlow:
		log.Infof("Shutdown NetFlow9/ipfix listener on %s:%d", s.hostname, s.port)
		state.Shutdown()
	case *utils.StateSFlow:
		log.Infof("Shutdown sFlow listener on %s:%d", s.hostname, s.port)
		state.Shutdown()
	case *utils.StateNFLegacy:
		log.Infof("Shutdown Netflow5 listener on %s:%d", s.hostname, s.port)
		state.Shutdown()
	default:
		log.Warnf("Unknown flow listener state type `%T` for %s:%d", s.hostname, s.port)
	}
}

// StartNetFlowRoutine returns a FlowRoutineState for StateNetFlow
func StartNetFlowRoutine(formatDriver format.FormatInterface, hostname string, port uint16) *FlowRoutineState {
	log.Info("Starting NetFlow9/ipfix listener...")
	state := &utils.StateNetFlow{
		Format: formatDriver,
		Logger: GetLogrusLevel(),
	}
	go func() {
		err := state.FlowRoutine(1, hostname, int(port), reusePort)
		if err != nil {
			log.Errorf("Error listener to netflow9/ipfix: %s", err)
		}
	}()
	return &FlowRoutineState{
		hostname: hostname,
		port:     port,
		state:    state,
	}
}

// StartSFlowRoutine returns a FlowRoutineState for StateSFlow
func StartSFlowRoutine(formatDriver format.FormatInterface, hostname string, port uint16) *FlowRoutineState {
	log.Info("Starting sFlow listener ...")
	state := &utils.StateSFlow{
		Format: formatDriver,
		Logger: GetLogrusLevel(),
	}
	go func() {
		err := state.FlowRoutine(1, hostname, int(port), reusePort)
		if err != nil {
			log.Errorf("Error listener to sflow: %s", err)
		}
	}()
	return &FlowRoutineState{
		hostname: hostname,
		port:     port,
		state:    state,
	}
}

// StartNFLegacyRoutine returns a FlowRoutineState for StateNFLegacy
func StartNFLegacyRoutine(formatDriver format.FormatInterface, hostname string, port uint16) *FlowRoutineState {
	log.Info("Starting NetFlow5 listener...")
	state := &utils.StateNFLegacy{
		Format: formatDriver,
		Logger: GetLogrusLevel(),
	}
	go func() {
		err := state.FlowRoutine(1, hostname, int(port), reusePort)
		if err != nil {
			log.Errorf("Error listener to netflow5: %s", err)
		}
	}()
	return &FlowRoutineState{
		hostname: hostname,
		port:     port,
		state:    state,
	}
}
