// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package config

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/DataDog/datadog-agent/pkg/util/fargate"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// Endpoint specifies an endpoint that the trace agent will write data (traces, stats & services) to.
type Endpoint struct {
	APIKey string `json:"-"` // never marshal this
	Host   string

	// NoProxy will be set to true when the proxy setting for the trace API endpoint
	// needs to be ignored (e.g. it is part of the "no_proxy" list in the yaml settings).
	NoProxy bool
}

// AgentConfig handles the interpretation of the configuration (with default
// behaviors) in one place. It is also a simple structure to share across all
// the Agent components, with 100% safe and reliable values.
// It is exposed with expvar, so make sure to exclude any sensible field
// from JSON encoding. Use New() to create an instance.
type AgentConfig struct {
	Enabled             bool
	FargateOrchestrator string // Fargate orchestrator, or empty if not Fargate

	// Global
	Hostname   string
	DefaultEnv string // the traces will default to this environment
	ConfigPath string // the source of this config, if any

	// Endpoints specifies the set of hosts and API keys where traces and stats
	// will be uploaded to. The first endpoint is the main configuration endpoint;
	// any following ones are read from the 'additional_endpoints' parts of the
	// configuration file, if present.
	Endpoints []*Endpoint

	// Concentrator
	BucketInterval   time.Duration // the size of our pre-aggregation per bucket
	ExtraAggregators []string

	// Sampler configuration
	ExtraSampleRate float64
	TargetTPS       float64
	MaxEPS          float64

	// Receiver
	ReceiverHost    string
	ReceiverPort    int
	ReceiverSocket  string // if not empty, UDS will be enabled on unix://<receiver_socket>
	ConnectionLimit int    // for rate-limiting, how many unique connections to allow in a lease period (30s)
	ReceiverTimeout int
	MaxRequestBytes int64 // specifies the maximum allowed request size for incoming trace payloads

	// Writers
	SynchronousFlushing     bool // Mode where traces are only submitted when FlushAsync is called, used for Serverless Extension
	StatsWriter             *WriterConfig
	TraceWriter             *WriterConfig
	ConnectionResetInterval time.Duration // frequency at which outgoing connections are reset. 0 means no reset is performed

	// internal telemetry
	StatsdHost   string
	StatsdPort   int
	StatsdPipe   string
	StatsdSocket string

	// logging
	LogLevel      string
	LogFilePath   string
	LogThrottling bool

	// watchdog
	MaxMemory        float64       // MaxMemory is the threshold (bytes allocated) above which program panics and exits, to be restarted
	MaxCPU           float64       // MaxCPU is the max UserAvg CPU the program should consume
	WatchdogInterval time.Duration // WatchdogInterval is the delay between 2 watchdog checks

	// http/s proxying
	ProxyURL          *url.URL
	SkipSSLValidation bool

	// filtering
	Ignore map[string][]string

	// ReplaceTags is used to filter out sensitive information from tag values.
	// It maps tag keys to a set of replacements. Only supported in A6.
	ReplaceTags []*ReplaceRule

	// GlobalTags list metadata that will be added to all spans
	GlobalTags map[string]string

	// transaction analytics
	AnalyzedRateByServiceLegacy map[string]float64
	AnalyzedSpansByService      map[string]map[string]float64

	// infrastructure agent binary
	DDAgentBin string

	// Obfuscation holds sensitive data obufscator's configuration.
	Obfuscation *ObfuscationConfig

	// RequireTags specifies a list of tags which must be present on the root span in order for a trace to be accepted.
	RequireTags []*Tag

	// RejectTags specifies a list of tags which must be absent on the root span in order for a trace to be accepted.
	RejectTags []*Tag

	// OTLPReceiver holds the configuration for OpenTelemetry receiver.
	OTLPReceiver *OTLP

	// ProxyTransportFunc returns the Transport's Proxy function to use.
	ProxyTransportFunc func() func(*http.Request) (*url.URL, error) `json:",omitempty"`
}

// OTLP holds the configuration for the OpenTelemetry receiver.
type OTLP struct {
	// BindHost specifies the host to bind the receiver to.
	BindHost string `mapstructure:"-"`

	// HTTPPort specifies the port to use for the plain HTTP receiver.
	// If unset (or 0), the receiver will be off.
	HTTPPort int `mapstructure:"http_port"`

	// GRPCPort specifies the port to use for the plain HTTP receiver.
	// If unset (or 0), the receiver will be off.
	GRPCPort int `mapstructure:"grpc_port"`

	// MaxRequestBytes specifies the maximum number of bytes that will be read
	// from an incoming HTTP request.
	MaxRequestBytes int64 `mapstructure:"-"`
}

// ObfuscationConfig holds the configuration for obfuscating sensitive data
// for various span types.
type ObfuscationConfig struct {
	// ES holds the obfuscation configuration for ElasticSearch bodies.
	ES JSONObfuscationConfig `mapstructure:"elasticsearch"`

	// Mongo holds the obfuscation configuration for MongoDB queries.
	Mongo JSONObfuscationConfig `mapstructure:"mongodb"`

	// SQLExecPlan holds the obfuscation configuration for SQL Exec Plans. This is strictly for safety related obfuscation,
	// not normalization. Normalization of exec plans is configured in SQLExecPlanNormalize.
	SQLExecPlan JSONObfuscationConfig `mapstructure:"sql_exec_plan"`

	// SQLExecPlanNormalize holds the normalization configuration for SQL Exec Plans.
	SQLExecPlanNormalize JSONObfuscationConfig `mapstructure:"sql_exec_plan_normalize"`

	// HTTP holds the obfuscation settings for HTTP URLs.
	HTTP HTTPObfuscationConfig `mapstructure:"http"`

	// RemoveStackTraces specifies whether stack traces should be removed.
	// More specifically "error.stack" tag values will be cleared.
	RemoveStackTraces bool `mapstructure:"remove_stack_traces"`

	// Redis holds the configuration for obfuscating the "redis.raw_command" tag
	// for spans of type "redis".
	Redis Enablable `mapstructure:"redis"`

	// Memcached holds the configuration for obfuscating the "memcached.command" tag
	// for spans of type "memcached".
	Memcached Enablable `mapstructure:"memcached"`
}

// HTTPObfuscationConfig holds the configuration settings for HTTP obfuscation.
type HTTPObfuscationConfig struct {
	// RemoveQueryStrings determines query strings to be removed from HTTP URLs.
	RemoveQueryString bool `mapstructure:"remove_query_string" json:"remove_query_string"`

	// RemovePathDigits determines digits in path segments to be obfuscated.
	RemovePathDigits bool `mapstructure:"remove_paths_with_digits" json:"remove_path_digits"`
}

// Enablable can represent any option that has an "enabled" boolean sub-field.
type Enablable struct {
	Enabled bool `mapstructure:"enabled"`
}

// JSONObfuscationConfig holds the obfuscation configuration for sensitive
// data found in JSON objects.
type JSONObfuscationConfig struct {
	// Enabled will specify whether obfuscation should be enabled.
	Enabled bool `mapstructure:"enabled"`

	// KeepValues will specify a set of keys for which their values will
	// not be obfuscated.
	KeepValues []string `mapstructure:"keep_values"`

	// ObfuscateSQLValues will specify a set of keys for which their values
	// will be passed through SQL obfuscation
	ObfuscateSQLValues []string `mapstructure:"obfuscate_sql_values"`
}

// ReplaceRule specifies a replace rule.
type ReplaceRule struct {
	// Name specifies the name of the tag that the replace rule addresses. However,
	// some exceptions apply such as:
	// • "resource.name" will target the resource
	// • "*" will target all tags and the resource
	Name string `mapstructure:"name"`

	// Pattern specifies the regexp pattern to be used when replacing. It must compile.
	Pattern string `mapstructure:"pattern"`

	// Re holds the compiled Pattern and is only used internally.
	Re *regexp.Regexp `mapstructure:"-"`

	// Repl specifies the replacement string to be used when Pattern matches.
	Repl string `mapstructure:"repl"`
}

// WriterConfig specifies configuration for an API writer.
type WriterConfig struct {
	// ConnectionLimit specifies the maximum number of concurrent outgoing
	// connections allowed for the sender.
	ConnectionLimit int `mapstructure:"connection_limit"`

	// QueueSize specifies the maximum number or payloads allowed to be queued
	// in the sender.
	QueueSize int `mapstructure:"queue_size"`

	// FlushPeriodSeconds specifies the frequency at which the writer's buffer
	// will be flushed to the sender, in seconds. Fractions are permitted.
	FlushPeriodSeconds float64 `mapstructure:"flush_period_seconds"`
}

// Tag represents a key/value pair.
type Tag struct {
	K, V string
}

// New returns a configuration with the default values.
func New() *AgentConfig {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	orch := fargate.GetOrchestrator(ctx)
	if orch == fargate.Unknown {
		orch = ""
	}
	cancel()
	if err := ctx.Err(); err != nil && err != context.Canceled {
		log.Errorf("Failed to get Fargate orchestrator. This may cause issues if you are in a Fargate instance: %v", err)
	}
	return &AgentConfig{
		Enabled:             true,
		FargateOrchestrator: string(orch),
		DefaultEnv:          "none",
		Endpoints:           []*Endpoint{{Host: "https://trace.agent.datadoghq.com"}},

		BucketInterval: time.Duration(10) * time.Second,

		ExtraSampleRate: 1.0,
		TargetTPS:       10,
		MaxEPS:          200,

		ReceiverHost:    "localhost",
		ReceiverPort:    8126,
		MaxRequestBytes: 50 * 1024 * 1024, // 50MB

		StatsWriter:             new(WriterConfig),
		TraceWriter:             new(WriterConfig),
		ConnectionResetInterval: 0, // disabled

		StatsdHost: "localhost",
		StatsdPort: 8125,

		LogLevel:      "INFO",
		LogFilePath:   DefaultLogFilePath,
		LogThrottling: true,

		MaxMemory:        5e8, // 500 Mb, should rarely go above 50 Mb
		MaxCPU:           0.5, // 50%, well behaving agents keep below 5%
		WatchdogInterval: 10 * time.Second,

		Ignore:                      make(map[string][]string),
		AnalyzedRateByServiceLegacy: make(map[string]float64),
		AnalyzedSpansByService:      make(map[string]map[string]float64),

		GlobalTags: make(map[string]string),

		DDAgentBin:   defaultDDAgentBin,
		OTLPReceiver: &OTLP{},

		ProxyTransportFunc: DefaultProxyTransportFunc,
	}
}

var DefaultProxyTransportFunc = func() func(*http.Request) (*url.URL, error) {
	return http.ProxyFromEnvironment
}

// APIKey returns the first (main) endpoint's API key.
func (c *AgentConfig) APIKey() string {
	if len(c.Endpoints) == 0 {
		return ""
	}
	return c.Endpoints[0].APIKey
}

// NewHTTPClient returns a new http.Client to be used for outgoing connections to the
// Datadog API.
func (c *AgentConfig) NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: c.NewHTTPTransport(),
	}
}

// NewHTTPTransport returns a new http.Transport to be used for outgoing connections to
// the Datadog API.
func (c *AgentConfig) NewHTTPTransport() *http.Transport {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: c.SkipSSLValidation},
		// below field values are from http.DefaultTransport (go1.12)
		Proxy: c.ProxyTransportFunc(),
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return transport
}
