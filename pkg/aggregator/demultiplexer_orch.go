// +build orchestrator

package aggregator

import (
	"github.com/DataDog/datadog-agent/pkg/forwarder"
	orch "github.com/DataDog/datadog-agent/pkg/orchestrator/config"
)

func buildOrchestratorForwarder() *forwarder.DefaultForwarder {
	return orch.NewOrchestratorForwarder()
}
