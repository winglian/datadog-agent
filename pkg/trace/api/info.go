// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package api

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/DataDog/datadog-agent/pkg/obfuscate"
	"github.com/DataDog/datadog-agent/pkg/trace/config/features"
	"github.com/DataDog/datadog-agent/pkg/trace/info"
)

// makeInfoHandler returns a new handler for handling the discovery endpoint.
func (r *HTTPReceiver) makeInfoHandler() (hash string, handler http.HandlerFunc) {
	var all []string
	for _, e := range endpoints {
		if e.IsEnabled != nil && !e.IsEnabled() {
			continue
		}
		if !e.Hidden {
			all = append(all, e.Pattern)
		}
	}
	type reducedConfig struct {
		DefaultEnv             string                        `json:"default_env"`
		TargetTPS              float64                       `json:"target_tps"`
		MaxEPS                 float64                       `json:"max_eps"`
		ReceiverPort           int                           `json:"receiver_port"`
		ReceiverSocket         string                        `json:"receiver_socket"`
		ConnectionLimit        int                           `json:"connection_limit"`
		ReceiverTimeout        int                           `json:"receiver_timeout"`
		MaxRequestBytes        int64                         `json:"max_request_bytes"`
		StatsdPort             int                           `json:"statsd_port"`
		MaxMemory              float64                       `json:"max_memory"`
		MaxCPU                 float64                       `json:"max_cpu"`
		AnalyzedSpansByService map[string]map[string]float64 `json:"analyzed_spans_by_service"`
		Obfuscation            obfuscate.Config              `json:"obfuscation"`
	}
	var oconf obfuscate.Config
	if o := r.conf.Obfuscation; o != nil {
		oconf = o.Export()
	}
	txt, err := json.MarshalIndent(struct {
		Version       string        `json:"version"`
		GitCommit     string        `json:"git_commit"`
		BuildDate     string        `json:"build_date"`
		Endpoints     []string      `json:"endpoints"`
		FeatureFlags  []string      `json:"feature_flags,omitempty"`
		ClientDropP0s bool          `json:"client_drop_p0s"`
		Config        reducedConfig `json:"config"`
	}{
		Version:       info.Version,
		GitCommit:     info.GitCommit,
		BuildDate:     info.BuildDate,
		Endpoints:     all,
		FeatureFlags:  features.All(),
		ClientDropP0s: true,
		Config: reducedConfig{
			DefaultEnv:             r.conf.DefaultEnv,
			TargetTPS:              r.conf.TargetTPS,
			MaxEPS:                 r.conf.MaxEPS,
			ReceiverPort:           r.conf.ReceiverPort,
			ReceiverSocket:         r.conf.ReceiverSocket,
			ConnectionLimit:        r.conf.ConnectionLimit,
			ReceiverTimeout:        r.conf.ReceiverTimeout,
			MaxRequestBytes:        r.conf.MaxRequestBytes,
			StatsdPort:             r.conf.StatsdPort,
			MaxMemory:              r.conf.MaxMemory,
			MaxCPU:                 r.conf.MaxCPU,
			AnalyzedSpansByService: r.conf.AnalyzedSpansByService,
			Obfuscation:            oconf,
		},
	}, "", "\t")
	if err != nil {
		panic(fmt.Errorf("Error making /info handler: %v", err))
	}
	h := sha256.Sum256(txt)
	return fmt.Sprintf("%x", h), func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, "%s", txt)
	}
}
