package goflowlib

import (
	"net"

	flowpb "github.com/netsampler/goflow2/pb"

	"github.com/DataDog/datadog-agent/pkg/netflow/common"
)

// ConvertFlow convert goflow flow structure to internal flow structure
func ConvertFlow(srcFlow *flowpb.FlowMessage) *common.Flow {
	return &common.Flow{
		ReceivedTimestamp: srcFlow.TimeReceived,
		StartTimestamp:    srcFlow.TimeFlowStart,
		EndTimestamp:      srcFlow.TimeFlowEnd,
		SrcAddr:           net.IP(srcFlow.SrcAddr).String(),
		DstAddr:           net.IP(srcFlow.DstAddr).String(),
		SrcPort:           srcFlow.SrcPort,
		DstPort:           srcFlow.DstPort,
		Proto:             srcFlow.Proto,
		Tos:               srcFlow.IPTos,
		InputInterface:    srcFlow.InIf,
		FlowType:          convertFlowType(srcFlow.Type),
		SamplerAddr:       net.IP(srcFlow.SamplerAddress).String(),
		OutputInterface:   srcFlow.OutIf,
		Direction:         srcFlow.FlowDirection,
		Bytes:             srcFlow.Bytes,
		Packets:           srcFlow.Packets,
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
