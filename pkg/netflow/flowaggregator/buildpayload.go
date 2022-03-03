package flowaggregator

import (
	"github.com/DataDog/datadog-agent/pkg/netflow/common"
	"github.com/DataDog/datadog-agent/pkg/netflow/payload"
)

func buildPayload(aggFlow *common.Flow) payload.FlowPayload {
	return payload.FlowPayload{
		// Keys
		SrcAddr:        aggFlow.SrcAddr,
		SrcPort:        aggFlow.SrcPort,
		DstAddr:        aggFlow.DstAddr,
		DstPort:        aggFlow.DstPort,
		Proto:          aggFlow.Proto,
		Tos:            aggFlow.Tos,
		InputInterface: aggFlow.InputInterface,

		// Non-Keys
		ReceivedTimestamp: aggFlow.ReceivedTimestamp,
		StartTimestamp:    aggFlow.StartTimestamp,
		EndTimestamp:      aggFlow.EndTimestamp,
		SamplerAddr:       aggFlow.SamplerAddr,
		FlowType:          string(aggFlow.FlowType),
		OutputInterface:   aggFlow.OutputInterface,
		Direction:         aggFlow.Direction,
		Bytes:             aggFlow.Bytes,
		Packets:           aggFlow.Packets,
	}
}
