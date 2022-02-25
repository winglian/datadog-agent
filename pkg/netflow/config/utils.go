package config

import "github.com/DataDog/datadog-agent/pkg/netflow/common"

func getDefaultPort(flowType common.FlowType) uint16 {
	switch flowType {
	case common.TypeIPFIX:
		return common.DefaultPortIPFIX
	case common.TypeNetFlow5, common.TypeNetFlow9:
		return common.DefaultPortNETFLOW
	case common.TypeSFlow5:
		return common.DefaultPortSFLOW
	}
	return 0
}
