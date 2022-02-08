// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021-present Datadog, Inc.

//go:build test
// +build test

package otlp

import (
	"bufio"
	"bytes"
	"fmt"
	"testing"

	"github.com/DataDog/datadog-agent/pkg/otlp/internal/testutil"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/cihub/seelog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsEnabled(t *testing.T) {
	tests := []struct {
		path    string
		enabled bool
	}{
		{path: "port/bindhost.yaml", enabled: true},
		{path: "port/disabled.yaml", enabled: false},
		{path: "port/invalid.yaml", enabled: true},
		{path: "port/nobindhost.yaml", enabled: true},

		{path: "receiver/noprotocols.yaml", enabled: true},
		{path: "receiver/portandreceiver.yaml", enabled: true},
		{path: "receiver/simple.yaml", enabled: true},
		{path: "receiver/advanced.yaml", enabled: true},
	}

	for _, testInstance := range tests {
		t.Run(testInstance.path, func(t *testing.T) {
			cfg, err := testutil.LoadConfig("./testdata/" + testInstance.path)
			require.NoError(t, err)
			assert.Equal(t, testInstance.enabled, IsEnabled(cfg))
		})
	}
}

func TestFromAgentConfigReceiver(t *testing.T) {
	tests := []struct {
		path string
		cfg  PipelineConfig
		err  string
	}{
		{
			path: "port/bindhost.yaml",
			cfg: PipelineConfig{
				OTLPReceiverConfig: testutil.OTLPConfigFromPorts("bindhost", 5678, 1234),
				TracePort:          5003,
				MetricsEnabled:     true,
				TracesEnabled:      true,
				Metrics: map[string]interface{}{
					"enabled":         true,
					"tag_cardinality": "low",
				},
			},
		},
		{
			path: "port/nobindhost.yaml",
			cfg: PipelineConfig{
				OTLPReceiverConfig: testutil.OTLPConfigFromPorts("localhost", 5678, 1234),
				TracePort:          5003,
				MetricsEnabled:     true,
				TracesEnabled:      true,
				Metrics: map[string]interface{}{
					"enabled":         true,
					"tag_cardinality": "low",
				},
			},
		},
		{
			path: "port/invalid.yaml",
			err:  fmt.Sprintf("internal trace port is invalid: -1 is out of [0, 65535] range"),
		},
		{
			path: "port/alldisabled.yaml",
			err:  "at least one OTLP signal needs to be enabled",
		},
		{
			path: "receiver/noprotocols.yaml",
			cfg: PipelineConfig{
				OTLPReceiverConfig: map[string]interface{}{},
				TracePort:          5003,
				MetricsEnabled:     true,
				TracesEnabled:      true,
				Metrics: map[string]interface{}{
					"enabled":         true,
					"tag_cardinality": "low",
				},
			},
		},
		{
			path: "receiver/portandreceiver.yaml",
			cfg: PipelineConfig{
				OTLPReceiverConfig: testutil.OTLPConfigFromPorts("localhost", 5679, 1234),
				TracePort:          5003,
				MetricsEnabled:     true,
				TracesEnabled:      true,
				Metrics: map[string]interface{}{
					"enabled":         true,
					"tag_cardinality": "low",
				},
			},
		},
		{
			path: "receiver/simple.yaml",
			cfg: PipelineConfig{
				OTLPReceiverConfig: map[string]interface{}{
					"protocols": map[string]interface{}{
						"grpc": nil,
						"http": nil,
					},
				},
				TracePort:      5003,
				MetricsEnabled: true,
				TracesEnabled:  true,
				Metrics: map[string]interface{}{
					"enabled":         true,
					"tag_cardinality": "low",
				},
			},
		},
		{
			path: "receiver/advanced.yaml",
			cfg: PipelineConfig{
				OTLPReceiverConfig: map[string]interface{}{
					"protocols": map[string]interface{}{
						"grpc": map[string]interface{}{
							"endpoint":               "0.0.0.0:5678",
							"max_concurrent_streams": 16,
							"transport":              "tcp",
							"keepalive": map[string]interface{}{
								"enforcement_policy": map[string]interface{}{
									"min_time": "10m",
								},
							},
						},
						"http": map[string]interface{}{
							"endpoint": "localhost:1234",
							"cors": map[string]interface{}{
								"allowed_origins": []interface{}{"http://test.com"},
								"allowed_headers": []interface{}{"ExampleHeader"},
							},
						},
					},
				},
				TracePort:      5003,
				MetricsEnabled: true,
				TracesEnabled:  true,
				Metrics: map[string]interface{}{
					"enabled":         true,
					"tag_cardinality": "low",
				},
			},
		},
	}

	for _, testInstance := range tests {
		t.Run(testInstance.path, func(t *testing.T) {
			cfg, err := testutil.LoadConfig("./testdata/" + testInstance.path)
			require.NoError(t, err)
			pcfg, err := FromAgentConfig(cfg)
			if err != nil || testInstance.err != "" {
				assert.Equal(t, testInstance.err, err.Error())
			} else {
				assert.Equal(t, testInstance.cfg, pcfg)
			}
		})
	}
}

func TestFromAgentConfigMetrics(t *testing.T) {
	tests := []struct {
		path string
		cfg  PipelineConfig
		err  string
	}{
		{
			path: "metrics/allconfig.yaml",
			cfg: PipelineConfig{
				OTLPReceiverConfig: testutil.OTLPConfigFromPorts("localhost", 5678, 1234),
				TracePort:          5003,
				MetricsEnabled:     true,
				TracesEnabled:      true,
				Metrics: map[string]interface{}{
					"delta_ttl":                                2400,
					"report_quantiles":                         false,
					"send_monotonic_counter":                   true,
					"resource_attributes_as_tags":              true,
					"instrumentation_library_metadata_as_tags": true,
					"tag_cardinality":                          "orchestrator",
					"histograms": map[string]interface{}{
						"mode":                   "counters",
						"send_count_sum_metrics": true,
					},
				},
			},
		},
	}

	for _, testInstance := range tests {
		t.Run(testInstance.path, func(t *testing.T) {
			cfg, err := testutil.LoadConfig("./testdata/" + testInstance.path)
			require.NoError(t, err)
			pcfg, err := FromAgentConfig(cfg)
			if err != nil || testInstance.err != "" {
				assert.Equal(t, testInstance.err, err.Error())
			} else {
				assert.Equal(t, testInstance.cfg, pcfg)
			}
		})
	}
}

func TestFromExperimentalEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		cfg  PipelineConfig
		err  string
	}{
		{
			name: "only gRPC",
			env: map[string]string{
				"DD_OTLP_GRPC_PORT": "4317",
			},
			cfg: PipelineConfig{
				OTLPReceiverConfig: map[string]interface{}{
					"protocols": map[string]interface{}{
						"grpc": map[string]interface{}{
							"endpoint": "localhost:4317",
						},
					},
				},
				MetricsEnabled: true,
				TracesEnabled:  true,
				TracePort:      5003,
				Metrics: map[string]interface{}{
					"enabled":         true,
					"tag_cardinality": "low",
				},
			},
		},
		{
			name: "only HTTP",
			env: map[string]string{
				"DD_OTLP_HTTP_PORT": "1234",
			},
			cfg: PipelineConfig{
				OTLPReceiverConfig: map[string]interface{}{
					"protocols": map[string]interface{}{
						"http": map[string]interface{}{
							"endpoint": "localhost:1234",
						},
					},
				},
				MetricsEnabled: true,
				TracesEnabled:  true,
				TracePort:      5003,
				Metrics: map[string]interface{}{
					"enabled":         true,
					"tag_cardinality": "low",
				},
			},
		},
		{
			name: "HTTP and gRPC",
			env: map[string]string{
				"DD_OTLP_GRPC_PORT": "4317",
				"DD_OTLP_HTTP_PORT": "1234",
			},
			cfg: PipelineConfig{
				OTLPReceiverConfig: map[string]interface{}{
					"protocols": map[string]interface{}{
						"grpc": map[string]interface{}{
							"endpoint": "localhost:4317",
						},
						"http": map[string]interface{}{
							"endpoint": "localhost:1234",
						},
					},
				},
				MetricsEnabled: true,
				TracesEnabled:  true,
				TracePort:      5003,
				Metrics: map[string]interface{}{
					"enabled":         true,
					"tag_cardinality": "low",
				},
			},
		},
	}

	for _, testInstance := range tests {
		t.Run(testInstance.name, func(t *testing.T) {
			// setup logging
			var b bytes.Buffer
			w := bufio.NewWriter(&b)
			l, err := seelog.LoggerFromWriterWithMinLevelAndFormat(w, seelog.DebugLvl, "[%LEVEL] %FuncShort: %Msg\n")
			require.NoError(t, err)
			log.SetupLogger(l, "warn")

			for env, val := range testInstance.env {
				t.Setenv(env, val)
			}
			cfg, err := testutil.LoadConfig("./testdata/empty.yaml")
			require.NoError(t, err)
			pcfg, err := FromAgentConfig(cfg)
			if err != nil || testInstance.err != "" {
				assert.Equal(t, testInstance.err, err.Error())
			} else {
				assert.Equal(t, testInstance.cfg, pcfg)
			}

			assert.Contains(t,
				b.String(),
				`OpenTelemetry OTLP receiver configuration is now public beta and has been moved out of the "experimental" section. `+
					`This section will be deprecated in a future Datadog Agent release. Please use the "otlp_config" section instead.`,
			)
		})
	}
}
