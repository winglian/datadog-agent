package common

import (
	"encoding/json"
	"hash/fnv"
	"strconv"
)

// Flow contains flow info used for aggreagtion
type Flow struct {
	FlowType          FlowType `json:"flow_type"`
	ReceivedTimestamp uint64   `json:"received_timestamp"`
	SamplingRate      uint64   `json:"sampling_rate"`
	Direction         uint32   `json:"direction"`

	// Sampler information
	SamplerAddr string `json:"sampler_addr"`

	// Flow time
	StartTimestamp uint64 `json:"start_timestamp"`
	EndTimestamp   uint64 `json:"end_timestamp"`

	// Size of the sampled packet
	Bytes   uint64 `json:"bytes"`
	Packets uint64 `json:"packets"`

	// Source/destination addresses
	SrcAddr string `json:"src_addr"` // FLOW KEY
	DstAddr string `json:"dst_addr"` // FLOW KEY

	// Layer 3 protocol (IPv4/IPv6/ARP/MPLS...)
	EtherType uint32 `json:"ether_type,omitempty"`

	// Layer 4 protocol
	Proto uint32 `json:"proto"` // FLOW KEY

	// Ports for UDP and TCP
	SrcPort uint32 `json:"src_port"` // FLOW KEY
	DstPort uint32 `json:"dst_port"` // FLOW KEY

	// SNMP Interface Index
	InputInterface  uint32 `json:"input_interface"` // FLOW KEY
	OutputInterface uint32 `json:"output_interface"`

	// Ethernet information

	Tos uint32 `json:"tos"` // FLOW KEY
}

// AggregationHash return a hash used as aggregation key
func (f *Flow) AggregationHash() string {
	h := fnv.New64()
	h.Write([]byte(f.SrcAddr))                           //nolint:errcheck
	h.Write([]byte(f.DstAddr))                           //nolint:errcheck
	h.Write([]byte(strconv.Itoa(int(f.SrcPort))))        //nolint:errcheck
	h.Write([]byte(strconv.Itoa(int(f.DstPort))))        //nolint:errcheck
	h.Write([]byte(strconv.Itoa(int(f.Proto))))          //nolint:errcheck
	h.Write([]byte(strconv.Itoa(int(f.Tos))))            //nolint:errcheck
	h.Write([]byte(strconv.Itoa(int(f.InputInterface)))) //nolint:errcheck
	return strconv.FormatUint(h.Sum64(), 16)
}

// AsJSONString returns a JSON string or "" in case of error during the Marshaling
// Used in debug logs. Marshalling to json can be costly if called in critical path.
func (f *Flow) AsJSONString() string {
	s, err := json.Marshal(f)
	if err != nil {
		return ""
	}
	return string(s)
}

// TelemetryTags return tags used for telemetry
func (f *Flow) TelemetryTags() []string {
	return []string{
		"sample_addr:" + f.SamplerAddr,
		"flow_type:" + string(f.FlowType),
	}
}
