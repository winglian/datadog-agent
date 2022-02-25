package flowaggregator

import (
	"sync"

	"github.com/DataDog/datadog-agent/pkg/util/log"

	"github.com/DataDog/datadog-agent/pkg/netflow/common"
)

type flowStore struct {
	flows map[string]*common.Flow
	mu    sync.Mutex
}

func newFlowStore() *flowStore {
	return &flowStore{
		flows: make(map[string]*common.Flow),
	}
}

func (f *flowStore) getFlows() []*common.Flow {
	f.mu.Lock()
	defer f.mu.Unlock()

	var flows []*common.Flow // init with optimal size
	for _, flow := range f.flows {
		flows = append(flows, flow)
	}

	// clear flows
	f.flows = make(map[string]*common.Flow)

	return flows
}

func (f *flowStore) addFlow(flowToAdd *common.Flow) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// TODO: handle port direction (see network-http-logger)
	// TODO: ignore ephemeral ports

	aggFlow, ok := f.flows[flowToAdd.AggregationHash()]
	log.Tracef("New Flow (digest=%s): %+v", flowToAdd.AggregationHash(), flowToAdd)
	if !ok {
		f.flows[flowToAdd.AggregationHash()] = flowToAdd
	} else {
		newAggFlow := *aggFlow
		newAggFlow.Bytes += flowToAdd.Bytes
		newAggFlow.Packets += flowToAdd.Packets
		log.Tracef("Existing Aggregated Flow (digest=%s): %+v", flowToAdd.AggregationHash(), aggFlow)
		log.Tracef("New Aggregated Flow (digest=%s): %+v", flowToAdd.AggregationHash(), newAggFlow)
		f.flows[flowToAdd.AggregationHash()] = &newAggFlow
	}
}
