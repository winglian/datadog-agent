// +build !orchestrator

package aggregator

import "github.com/DataDog/datadog-agent/pkg/forwarder"

func buildOrchestratorForwarder() *forwarder.DefaultForwarder {
	return nil
}
