// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package encoding

import (
	"strings"
	"sync"

	model "github.com/DataDog/agent-payload/v5/process"
	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/network"
	"github.com/DataDog/datadog-agent/pkg/network/http"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/datadog-agent/pkg/process/util"
	"github.com/gogo/protobuf/jsonpb"
)

var (
	pSerializer = protoSerializer{}
	jSerializer = jsonSerializer{
		marshaller: jsonpb.Marshaler{
			EmitDefaults: true,
		},
	}

	cfgOnce  = sync.Once{}
	agentCfg *model.AgentConfiguration
)

// Marshaler is an interface implemented by all Connections serializers
type Marshaler interface {
	Marshal(conns *network.Connections) ([]byte, error)
	ContentType() string
}

// Unmarshaler is an interface implemented by all Connections deserializers
type Unmarshaler interface {
	Unmarshal([]byte) (*model.Connections, error)
}

// GetMarshaler returns the appropriate Marshaler based on the given accept header
func GetMarshaler(accept string) Marshaler {
	if strings.Contains(accept, ContentTypeProtobuf) {
		return pSerializer
	}

	return jSerializer
}

// GetUnmarshaler returns the appropriate Unmarshaler based on the given content type
func GetUnmarshaler(ctype string) Unmarshaler {
	if strings.Contains(ctype, ContentTypeProtobuf) {
		return pSerializer
	}

	return jSerializer
}

func modelConnections(conns *network.Connections) *model.Connections {
	cfgOnce.Do(func() {
		agentCfg = &model.AgentConfiguration{
			NpmEnabled: config.Datadog.GetBool("network_config.enabled"),
			TsmEnabled: config.Datadog.GetBool("service_monitoring_config.enabled"),
		}
	})

	agentConns := make([]*model.Connection, len(conns.Conns))
	routeIndex := make(map[string]RouteIdx)
	httpIndex, tagsIndex := FormatHTTPStats(conns.HTTP)
	httpMatches := make(map[http.Key]struct{}, len(httpIndex))
	ipc := make(ipCache, len(conns.Conns)/2)
	dnsFormatter := newDNSFormatter(conns, ipc)
	tagsSet := network.NewTagsSet()

	for i, conn := range conns.Conns {
		var httpAggregations *model.HTTPAggregations

		httpKeys := httpKeysFromConn(conn)
		for _, httpKey := range httpKeys {
			httpAggregations := httpIndex[httpKey]
			if httpAggregations != nil {
				httpMatches[httpKey] = struct{}{}
				conn.Tags |= tagsIndex[httpKey]
				delete(httpIndex, httpKey);
				break
			}
		}

		agentConns[i] = FormatConnection(conn, routeIndex, httpAggregations, dnsFormatter, ipc, tagsSet)
	}

	if len(httpIndex) > 0 {
		log.Infof("***** Printing orphans ******")
		for k, v := range httpIndex {
			var saddr, daddr util.Address
			if k.SrcIPHigh == 0 && k.DstIPHigh == 0 {
				saddr = util.V4Address(uint32(k.SrcIPLow))
				daddr = util.V4Address(uint32(k.DstIPLow))
			} else {
				saddr = util.V6Address(k.SrcIPLow, k.SrcIPHigh)
				daddr = util.V6Address(k.DstIPLow, k.DstIPHigh)
			}
			log.Infof("  %v:%v -> %v:%v, %v aggregations",
				saddr,
				k.SrcPort,
				daddr,
				k.DstPort,
				len(v.EndpointAggregations),
			)
		}
	}

	routes := make([]*model.Route, len(routeIndex))
	for _, v := range routeIndex {
		routes[v.Idx] = &v.Route
	}

	payload := new(model.Connections)
	payload.AgentConfiguration = agentCfg
	payload.Conns = agentConns
	payload.Domains = dnsFormatter.Domains()
	payload.Dns = dnsFormatter.DNS()
	payload.ConnTelemetry = FormatConnTelemetry(conns.ConnTelemetry)
	payload.CompilationTelemetryByAsset = FormatCompilationTelemetry(conns.CompilationTelemetryByAsset)
	payload.Routes = routes
	payload.Tags = tagsSet.GetStrings()

	return payload
}
