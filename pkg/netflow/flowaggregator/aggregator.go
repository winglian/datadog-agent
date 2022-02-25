package flowaggregator

import (
	"encoding/json"
	"time"

	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/epforwarder"
	"github.com/DataDog/datadog-agent/pkg/util/log"

	"github.com/DataDog/datadog-agent/pkg/netflow/common"
	"github.com/DataDog/datadog-agent/pkg/netflow/config"
)

// FlowAggregator is used for space and time aggregation of NetFlow flows
type FlowAggregator struct {
	flowIn        chan *common.Flow // TODO: change to lighter struct than flowpb.FlowMessage ?
	flushInterval time.Duration
	flowStore     *flowStore
	sender        aggregator.Sender
	stopChan      chan struct{}
	logPayload    bool
}

// GetFlowInChan returns flow input chan
func (agg *FlowAggregator) GetFlowInChan() chan *common.Flow {
	return agg.flowIn
}

// NewFlowAggregator returns a new FlowAggregator
func NewFlowAggregator(sender aggregator.Sender, config *config.NetflowConfig) *FlowAggregator {
	return &FlowAggregator{
		flowIn:        make(chan *common.Flow, config.AggregatorBufferSize),
		flowStore:     newFlowStore(),
		flushInterval: time.Duration(config.FlushInterval) * time.Second,
		sender:        sender,
		stopChan:      make(chan struct{}),
		logPayload:    config.LogPayloads,
	}
}

func (agg *FlowAggregator) flush(flushTime time.Time) {
	flows := agg.flowStore.getFlows()
	if len(flows) == 0 {
		return
	}
	// TODO: Add flush count telemetry e.g. aggregator newFlushCountStats()

	// For debug purposes print out all flows
	if agg.logPayload {
		log.Debug("Flushing the following Events:")
		for _, flow := range flows {
			log.Debugf("%s", flow)
		}
	}
	agg.sendFlows(flows)
}

func (agg *FlowAggregator) sendFlows(flows []*common.Flow) {
	for _, flow := range flows {
		flowPayload := buildPayload(flow)
		payloadBytes, err := json.Marshal(flowPayload)
		if err != nil {
			log.Errorf("Error marshalling device metadata: %s", err)
			continue
		}
		agg.sender.EventPlatformEvent(string(payloadBytes), epforwarder.EventTypeNetworkDevicesNetFlow)
	}
}

// Start will start the FlowAggregator worker
func (agg *FlowAggregator) Start() {
	log.Info("Flow Aggregator started")
	go agg.run()
	agg.flushLoop() // blocking call
}

func (agg *FlowAggregator) run() {
	for {
		select {
		case <-agg.stopChan:
			log.Info("Stopping aggregator")
			return
		case flow := <-agg.flowIn:
			log.Debugf("Received new flow")
			agg.flowStore.addFlow(flow)
		}
	}
}

// Stop will stop running FlowAggregator
func (agg *FlowAggregator) Stop() {
	agg.stopChan <- struct{}{}
}

func (agg *FlowAggregator) flushLoop() {
	var flushTicker <-chan time.Time

	if agg.flushInterval > 0 {
		flushTicker = time.NewTicker(agg.flushInterval).C
	} else {
		// TODO: validate that flush interval is positive?
		log.Debugf("flushInterval set to 0: will never flush automatically")
	}

	for {
		select {
		// stop sequence
		case <-agg.stopChan:
			return
		// automatic flush sequence
		case t := <-flushTicker:
			agg.flush(t)
		}
	}
}
