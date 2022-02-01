// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package dockerfile

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/DataDog/datadog-agent/pkg/logs/internal/parsers"
	parsertesting "github.com/DataDog/datadog-agent/pkg/logs/internal/parsers/internal/testing"
	"github.com/DataDog/datadog-agent/pkg/logs/message"
)

func TestParser(t *testing.T) {
	pt := parsertesting.NewParserTester(New())
	defer pt.Stop()

	pt.SendLine([]byte(`{"log":"a full message\n","stream":"stderr","time":"2019-06-06T16:35:55.930852911Z"}`))
	msg := pt.GetMessage()
	assert.Equal(t, []byte("a full message"), msg.Content)
	assert.Equal(t, message.StatusError, msg.Status)
	assert.Equal(t, "2019-06-06T16:35:55.930852911Z", msg.Timestamp)

	pt.SendLine([]byte(`{"log":"a message","stream":"stderr","time":"2019-06-06T16:35:55.930852911Z"}`))
	pt.SendLine([]byte(`{"log":"a second message","stream":"stdout","time":"2019-06-06T16:35:55.930852912Z"}`))
	pt.SendLine([]byte(`{"log":"a third message\n","stream":"stdout","time":"2019-06-06T16:40:55.930852913Z"}`))

	msg = pt.GetMessage()
	assert.Equal(t, []byte("a messagea second messagea third message"), msg.Content)
	assert.Equal(t, message.StatusInfo, msg.Status)                  // from the last message
	assert.Equal(t, "2019-06-06T16:40:55.930852913Z", msg.Timestamp) // from the last message
}

func TestProcess(t *testing.T) {
	var (
		msg     parsers.Message
		partial bool
		err     error
	)

	parser := dockerFileFormat{}
	msg, partial, err = parser.Process([]byte(`{"log":"a message","stream":"stderr","time":"2019-06-06T16:35:55.930852911Z"}`))
	assert.Nil(t, err)
	assert.True(t, partial)
	assert.Equal(t, []byte("a message"), msg.Content)
	assert.Equal(t, message.StatusError, msg.Status)
	assert.Equal(t, "2019-06-06T16:35:55.930852911Z", msg.Timestamp)

	parser = dockerFileFormat{}
	msg, partial, err = parser.Process([]byte(`{"log":"a second message","stream":"stdout","time":"2019-06-06T16:35:55.930852912Z"}`))
	assert.Nil(t, err)
	assert.True(t, partial)
	assert.Equal(t, []byte("a second message"), msg.Content)
	assert.Equal(t, message.StatusInfo, msg.Status)
	assert.Equal(t, "2019-06-06T16:35:55.930852912Z", msg.Timestamp)

	parser = dockerFileFormat{}
	msg, partial, err = parser.Process([]byte(`{"log":"a third message\n","stream":"stdout","time":"2019-06-06T16:35:55.930852913Z"}`))
	assert.Nil(t, err)
	assert.False(t, partial)
	assert.Equal(t, []byte("a third message"), msg.Content)
	assert.Equal(t, message.StatusInfo, msg.Status)
	assert.Equal(t, "2019-06-06T16:35:55.930852913Z", msg.Timestamp)

	parser = dockerFileFormat{}
	msg, partial, err = parser.Process([]byte("a wrong message"))
	assert.NotNil(t, err)
	assert.False(t, partial)
	assert.Equal(t, []byte("a wrong message"), msg.Content)
	assert.Equal(t, message.StatusInfo, msg.Status)

	parser = dockerFileFormat{}
	msg, partial, err = parser.Process([]byte(`{"log":"","stream":"stdout","time":"2019-06-06T16:35:55.930852914Z"}`))
	assert.Nil(t, err)
	assert.False(t, partial)
	assert.Equal(t, []byte(""), msg.Content)
	assert.Equal(t, message.StatusInfo, msg.Status)
	assert.Equal(t, "2019-06-06T16:35:55.930852914Z", msg.Timestamp)

	parser = dockerFileFormat{}
	msg, partial, err = parser.Process([]byte(`{"log":"\n","stream":"stdout","time":"2019-06-06T16:35:55.930852915Z"}`))
	assert.Nil(t, err)
	assert.False(t, partial)
	assert.Equal(t, []byte(""), msg.Content)
	assert.Equal(t, message.StatusInfo, msg.Status)
	assert.Equal(t, "2019-06-06T16:35:55.930852915Z", msg.Timestamp)
}
