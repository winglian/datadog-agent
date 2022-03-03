package flowaggregator

import (
	"github.com/DataDog/datadog-agent/pkg/netflow/common"
	"github.com/DataDog/datadog-agent/pkg/netflow/payload"
)

func buildPayload(aggFlow *common.Flow) payload.FlowPayload {
	return payload.FlowPayload{
		FlowType:          string(aggFlow.FlowType),
		ReceivedTimestamp: aggFlow.ReceivedTimestamp,
		SamplingRate:      aggFlow.SamplingRate,
		Direction:         aggFlow.Direction,
		SamplerAddr:       aggFlow.SamplerAddr,
		StartTimestamp:    aggFlow.StartTimestamp,
		EndTimestamp:      aggFlow.EndTimestamp,
		Bytes:             aggFlow.Bytes,
		Packets:           aggFlow.Packets,
		SrcAddr:           aggFlow.SrcAddr,
		DstAddr:           aggFlow.DstAddr,
		EtherType:         aggFlow.EtherType,
		Proto:             aggFlow.Proto,
		SrcPort:           aggFlow.SrcPort,
		DstPort:           aggFlow.DstPort,
		InputInterface:    aggFlow.InputInterface,
		OutputInterface:   aggFlow.OutputInterface,
		Tos:               aggFlow.Tos,
	}
}
