package common

import (
	"encoding/json"
	"hash/fnv"
	"strconv"
)

// Flow contains flow info used for aggreagtion
type Flow struct {
	// Flow Keys
	SrcAddr        string `json:"src_addr"`
	DstAddr        string `json:"dst_addr"`
	SrcPort        uint32 `json:"src_port"`
	DstPort        uint32 `json:"dst_port"`
	Proto          uint32 `json:"proto"`
	Tos            uint32 `json:"tos"`
	InputInterface uint32 `json:"input_interface"`

	// Non Keys
	ReceivedTimestamp uint64   `json:"received_timestamp"`
	FlowType          FlowType `json:"flow_type"`
	SamplerAddr       string   `json:"sampler_addr"`
	OutputInterface   uint32   `json:"output_interface"`
	Direction         uint32   `json:"direction"`
	Bytes             uint64   `json:"bytes"`
	Packets           uint64   `json:"packets"`
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
