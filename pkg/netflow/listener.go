package netflow

import (
	"fmt"

	"github.com/DataDog/datadog-agent/pkg/netflow/common"
	"github.com/DataDog/datadog-agent/pkg/netflow/config"
	"github.com/DataDog/datadog-agent/pkg/netflow/flowaggregator"
	"github.com/DataDog/datadog-agent/pkg/netflow/goflowlib"
)

// netflowListener contains state of goflow listener and the related netflow config
// flowState can be of type *utils.StateNetFlow/StateSFlow/StateNFLegacy
type netflowListener struct {
	flowState *goflowlib.FlowRoutineState
	config    config.ListenerConfig
}

// Shutdown will close the goflow listener state
func (l *netflowListener) shutdown() {
	l.flowState.Shutdown()
}

func startFlowListener(listenerConfig config.ListenerConfig, flowAgg *flowaggregator.FlowAggregator) (*netflowListener, error) {
	var flowState *goflowlib.FlowRoutineState

	formatDriver := goflowlib.NewAggregatorFormatDriver(flowAgg.GetFlowInChan())

	switch listenerConfig.FlowType {
	case common.TypeNetFlow9, common.TypeIPFIX:
		flowState = goflowlib.StartNetFlowRoutine(formatDriver, listenerConfig.BindHost, listenerConfig.Port)
	case common.TypeSFlow5:
		flowState = goflowlib.StartSFlowRoutine(formatDriver, listenerConfig.BindHost, listenerConfig.Port)
	case common.TypeNetFlow5:
		flowState = goflowlib.StartNFLegacyRoutine(formatDriver, listenerConfig.BindHost, listenerConfig.Port)
	default:
		return nil, fmt.Errorf("unknown flow type: %s", listenerConfig.FlowType)
	}

	return &netflowListener{
		flowState: flowState,
		config:    listenerConfig,
	}, nil
}
