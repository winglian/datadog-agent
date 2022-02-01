// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package noop

import (
	"testing"

	parsertesting "github.com/DataDog/datadog-agent/pkg/logs/internal/parsers/internal/testing"
	"github.com/stretchr/testify/assert"
)

func TestNoopParserHandleMessages(t *testing.T) {
	pt := parsertesting.NewParserTester(New())
	defer pt.Stop()

	pt.SendLine([]byte("Foo"))
	msg := pt.GetMessage()
	assert.False(t, msg.IsPartial)
	assert.Equal(t, []byte("Foo"), msg.Content)
}
