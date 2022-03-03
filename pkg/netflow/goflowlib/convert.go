package goflowlib

import (
	"net"

	flowpb "github.com/netsampler/goflow2/pb"

	"github.com/DataDog/datadog-agent/pkg/netflow/common"
)

// ConvertFlow convert goflow flow structure to internal flow structure
func ConvertFlow(srcFlow *flowpb.FlowMessage) *common.Flow {
	return &common.Flow{
		FlowType:          convertFlowType(srcFlow.Type),
		ReceivedTimestamp: srcFlow.TimeReceived,
		SamplingRate:      srcFlow.SamplingRate,
		Direction:         srcFlow.FlowDirection,
		SamplerAddr:       net.IP(srcFlow.SamplerAddress).String(),
		StartTimestamp:    srcFlow.TimeFlowStart,
		EndTimestamp:      srcFlow.TimeFlowEnd,
		Bytes:             srcFlow.Bytes,
		Packets:           srcFlow.Packets,
		SrcAddr:           net.IP(srcFlow.SrcAddr).String(),
		DstAddr:           net.IP(srcFlow.DstAddr).String(),
		EtherType:         srcFlow.Etype,
		Proto:             srcFlow.Proto,
		SrcPort:           srcFlow.SrcPort,
		DstPort:           srcFlow.DstPort,
		InputInterface:    srcFlow.InIf,
		OutputInterface:   srcFlow.OutIf,
		Tos:               srcFlow.IPTos,
	}
}

func convertFlowType(flowType flowpb.FlowMessage_FlowType) common.FlowType {
	var flowTypeStr common.FlowType
	switch flowType {
	case flowpb.FlowMessage_SFLOW_5:
		flowTypeStr = common.TypeSFlow5
	case flowpb.FlowMessage_NETFLOW_V5:
		flowTypeStr = common.TypeNetFlow5
	case flowpb.FlowMessage_NETFLOW_V9:
		flowTypeStr = common.TypeNetFlow9
	case flowpb.FlowMessage_IPFIX:
		flowTypeStr = common.TypeIPFIX
	default:
		flowTypeStr = common.TypeUnknown
	}
	return flowTypeStr
}
