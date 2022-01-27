// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package channel

import (
	"fmt"
	"testing"

	"github.com/DataDog/datadog-agent/pkg/logs/config"
	"github.com/DataDog/datadog-agent/pkg/logs/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeServiceName(t *testing.T) {
	assert.Equal(t, "agent", computeServiceName(nil, "toto"))
	lambdaConfig := &config.Lambda{}
	assert.Equal(t, "my-service-name", computeServiceName(lambdaConfig, "my-service-name"))
	assert.Equal(t, "my-service-name", computeServiceName(lambdaConfig, "MY-SERVICE-NAME"))
	assert.Equal(t, "", computeServiceName(lambdaConfig, ""))
}

func TestRun(t *testing.T) {
	source := config.NewLogSource("test", &config.LogsConfig{})
	input := make(chan *config.ChannelMessage, 1)
	output := make(chan *message.Message, 1)
	tailer := NewTailer(source, input, output)

	require.Equal(t, tailer.Identifier(), fmt.Sprintf("channel:%p", input))

	tailer.Start()
	input <- &config.ChannelMessage{Content: []byte("hello")}
	got := <-output
	require.Equal(t, []byte("hello"), got.Content)
	tailer.Stop()
}
