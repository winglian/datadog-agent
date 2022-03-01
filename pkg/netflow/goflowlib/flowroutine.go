package goflowlib

import (
	"fmt"

	"github.com/netsampler/goflow2/format"
	"github.com/netsampler/goflow2/utils"

	"github.com/DataDog/datadog-agent/pkg/util/log"

	"github.com/DataDog/datadog-agent/pkg/netflow/common"
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

// StartFlowRoutine starts one of the goflow flow routine depending on the flow type
func StartFlowRoutine(flowType common.FlowType, bindHost string, port uint16, flowInChan chan *common.Flow) (*FlowRoutineState, error) {
	var flowState *FlowRoutineState
	formatDriver := NewAggregatorFormatDriver(flowInChan)

	switch flowType {
	case common.TypeNetFlow9, common.TypeIPFIX:
		flowState = startNetFlowRoutine(formatDriver, bindHost, port)
	case common.TypeSFlow5:
		flowState = startSFlowRoutine(formatDriver, bindHost, port)
	case common.TypeNetFlow5:
		flowState = startNFLegacyRoutine(formatDriver, bindHost, port)
	default:
		return nil, fmt.Errorf("unknown flow type: %s", flowType)
	}
	return flowState, nil
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

// startNetFlowRoutine returns a FlowRoutineState for StateNetFlow
func startNetFlowRoutine(formatDriver format.FormatInterface, hostname string, port uint16) *FlowRoutineState {
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

// startSFlowRoutine returns a FlowRoutineState for StateSFlow
func startSFlowRoutine(formatDriver format.FormatInterface, hostname string, port uint16) *FlowRoutineState {
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

// startNFLegacyRoutine returns a FlowRoutineState for StateNFLegacy
func startNFLegacyRoutine(formatDriver format.FormatInterface, hostname string, port uint16) *FlowRoutineState {
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
