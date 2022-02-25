// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020-present Datadog, Inc.

package config

import (
	"fmt"

	coreconfig "github.com/DataDog/datadog-agent/pkg/config"

	"github.com/DataDog/datadog-agent/pkg/netflow/common"
)

// NetflowConfig contains configuration for NetFlow collector.
type NetflowConfig struct {
	Listeners            []ListenerConfig `mapstructure:"listeners"`
	StopTimeout          int              `mapstructure:"stop_timeout"`
	AggregatorBufferSize int              `mapstructure:"aggregator_buffer_size"`
	LogPayloads          bool             `mapstructure:"log_payloads"`
	FlushInterval        int              `mapstructure:"flush_interval"`
}

// ListenerConfig contains configuration for a single flow listener
type ListenerConfig struct {
	FlowType common.FlowType `mapstructure:"flow_type"`
	Port     uint16          `mapstructure:"port"`
	BindHost string          `mapstructure:"bind_host"`

	// TODO: remove after dev stage
	SendEvents  bool `mapstructure:"send_events"`
	SendMetrics bool `mapstructure:"send_metrics"`
}

// ReadConfig builds and returns configuration from Agent configuration.
func ReadConfig() (*NetflowConfig, error) {
	var mainConfig NetflowConfig

	err := coreconfig.Datadog.UnmarshalKey("network_devices.netflow", &mainConfig)
	if err != nil {
		return nil, err
	}
	for _, listenerConfig := range mainConfig.Listeners {
		if listenerConfig.Port == 0 {
			listenerConfig.Port = getDefaultPort(listenerConfig.FlowType)
			if listenerConfig.Port == 0 {
				return nil, fmt.Errorf("no default port found for `%s`, a valid port must be set", listenerConfig.FlowType)
			}
		}
		if listenerConfig.BindHost == "" {
			// Default to global bind_host option.
			listenerConfig.BindHost = coreconfig.GetBindHost()
		}
	}

	if mainConfig.StopTimeout == 0 {
		mainConfig.StopTimeout = common.DefaultStopTimeout
	}

	return &mainConfig, nil
}

// Addr returns the host:port address to listen on.
func (c *ListenerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.BindHost, c.Port)
}
