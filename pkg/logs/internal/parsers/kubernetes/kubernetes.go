// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package kubernetes

import (
	"bytes"
	"errors"

	"github.com/DataDog/datadog-agent/pkg/logs/internal/parsers"
	"github.com/DataDog/datadog-agent/pkg/logs/internal/parsers/internal/base"
	"github.com/DataDog/datadog-agent/pkg/logs/message"
)

var (
	// one-space delimiter, created outside of a hot loop
	spaceByte = []byte{' '}
)

// New creates a new parser that parses Kubernetes-formatted log lines.
//
// Kubernetes log lines follow the pattern '<timestamp> <stream> <flag> <content>'; see
// https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/kuberuntime/logs/logs.go.
//
// For example: `2018-09-20T11:54:11.753589172Z stdout F This is my message`
func New() parsers.Parser {
	p := &kubernetesFormat{}
	p.CombiningParserBase.Process = p.Process
	return p
}

type kubernetesFormat struct {
	base.CombiningParserBase
}

// Process handles the actual parsing
func (p *kubernetesFormat) Process(data []byte) (parsers.Message, bool, error) {
	var status = message.StatusInfo
	var flag string
	var timestamp string
	// split '<timestamp> <stream> <flag> <content>' into its components
	components := bytes.SplitN(data, spaceByte, 4)
	if len(components) < 3 {
		return parsers.Message{
			Content:   data,
			Status:    status,
			Timestamp: timestamp,
		}, isPartial(flag), errors.New("cannot parse the log line")
	}
	var content []byte
	if len(components) > 3 {
		content = components[3]
	}
	status = getStatus(components[1])
	timestamp = string(components[0])
	flag = string(components[2])

	return parsers.Message{
		Content:   content,
		Status:    status,
		Timestamp: timestamp,
	}, isPartial(flag), nil
}

func isPartial(flag string) bool {
	return flag == "P"
}

// getStatus returns the status of the message based on
// the value of the STREAM_TYPE field in the header,
// returns the status INFO by default
func getStatus(streamType []byte) string {
	switch string(streamType) {
	case "stdout":
		return message.StatusInfo
	case "stderr":
		return message.StatusError
	default:
		return message.StatusInfo
	}
}
