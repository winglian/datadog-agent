// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package payload

// FlowPayload contains network devices flows
type FlowPayload struct {
	FlowType          string `json:"flow_type"`
	ReceivedTimestamp uint64 `json:"received_timestamp"`
	SamplingRate      uint64 `json:"sampling_rate"`
	Direction         uint32 `json:"direction"`
	SamplerAddr       string `json:"sampler_addr"`
	StartTimestamp    uint64 `json:"start_timestamp"`
	EndTimestamp      uint64 `json:"end_timestamp"`
	Bytes             uint64 `json:"bytes"`
	Packets           uint64 `json:"packets"`
	SrcAddr           string `json:"src_addr"`
	DstAddr           string `json:"dst_addr"`
	EtherType         uint32 `json:"ether_type"`
	Proto             uint32 `json:"proto"`
	SrcPort           uint32 `json:"src_port"`
	DstPort           uint32 `json:"dst_port"`
	InputInterface    uint32 `json:"input_interface"`
	OutputInterface   uint32 `json:"output_interface"`
	Tos               uint32 `json:"Tos"`
}
