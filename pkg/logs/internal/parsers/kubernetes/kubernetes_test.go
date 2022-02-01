// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package kubernetes

import (
	"testing"

	parsertesting "github.com/DataDog/datadog-agent/pkg/logs/internal/parsers/internal/testing"
	"github.com/DataDog/datadog-agent/pkg/logs/message"
	"github.com/stretchr/testify/assert"
)

var containerdHeaderOut = "2018-09-20T11:54:11.753589172Z stdout F"
var partialContainerdHeaderOut = "2018-09-20T11:54:11.753589172Z stdout P"

func TestKubernetesGetStatus(t *testing.T) {
	assert.Equal(t, message.StatusInfo, getStatus([]byte("stdout")))
	assert.Equal(t, message.StatusError, getStatus([]byte("stderr")))
	assert.Equal(t, message.StatusInfo, getStatus([]byte("")))
}

func TestParser(t *testing.T) {
	pt := parsertesting.NewParserTester(New())
	defer pt.Stop()

	pt.SendLine([]byte(partialContainerdHeaderOut + " " + "part1"))
	pt.SendLine([]byte(containerdHeaderOut + " " + "part2"))
	msg := pt.GetMessage()
	assert.Equal(t, message.StatusInfo, msg.Status)
	assert.Equal(t, []byte("part1part2"), msg.Content)
}

func TestKubernetesParserShouldSucceedWithValidInput(t *testing.T) {
	validMessage := containerdHeaderOut + " " + "anything"
	msg, partial, err := (&kubernetesFormat{}).Process([]byte(validMessage))
	assert.Nil(t, err)
	assert.False(t, partial)
	assert.Equal(t, message.StatusInfo, msg.Status)
	assert.Equal(t, []byte("anything"), msg.Content)
}

func TestKubernetesParserShouldSucceedWithPartialFlag(t *testing.T) {
	validMessage := partialContainerdHeaderOut + " " + "anything"
	msg, partial, err := (&kubernetesFormat{}).Process([]byte(validMessage))
	assert.Nil(t, err)
	assert.True(t, partial)
	assert.Equal(t, message.StatusInfo, msg.Status)
	assert.Equal(t, []byte("anything"), msg.Content)
}

func TestKubernetesParserShouldHandleEmptyMessage(t *testing.T) {
	msg, partial, err := (&kubernetesFormat{}).Process([]byte(containerdHeaderOut))
	assert.Nil(t, err)
	assert.Equal(t, 0, len(msg.Content))
	assert.False(t, partial)
	assert.Equal(t, message.StatusInfo, msg.Status)
	assert.Equal(t, "2018-09-20T11:54:11.753589172Z", msg.Timestamp)
}

func TestKubernetesParserShouldFailWithInvalidInput(t *testing.T) {
	// Only timestamp
	var err error
	log := []byte("2018-09-20T11:54:11.753589172Z foo")
	msg, partial, err := (&kubernetesFormat{}).Process(log)
	assert.False(t, partial)
	assert.NotNil(t, err)
	assert.Equal(t, log, msg.Content)
	assert.Equal(t, message.StatusInfo, msg.Status)
	assert.Equal(t, "", msg.Timestamp)

	// Missing timestamp but with 3 spaces, the message is valid
	// FIXME: We might want to handle that
	log = []byte("stdout F foo bar")
	msg, partial, err = (&kubernetesFormat{}).Process(log)
	assert.Nil(t, err)
}
