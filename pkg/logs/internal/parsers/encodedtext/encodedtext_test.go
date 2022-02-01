// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package encodedtext

import (
	"testing"

	parsertesting "github.com/DataDog/datadog-agent/pkg/logs/internal/parsers/internal/testing"
	"github.com/stretchr/testify/assert"
)

func TestUTF16LEParserHandleMessages(t *testing.T) {
	pt := parsertesting.NewParserTester(New(UTF16LE))
	defer pt.Stop()
	testMsg := []byte{'F', 0x0, 'o', 0x0, 'o', 0x0}
	pt.SendLine(testMsg)
	msg := pt.GetMessage()
	assert.Equal(t, "Foo", string(msg.Content))

	// We should support BOM
	testMsg = []byte{0xFF, 0xFE, 'F', 0x0, 'o', 0x0, 'o', 0x0}
	pt.SendLine(testMsg)
	msg = pt.GetMessage()
	assert.Equal(t, "Foo", string(msg.Content))

	// BOM overrides endianness
	testMsg = []byte{0xFE, 0xFF, 0x0, 'F', 0x0, 'o', 0x0, 'o'}
	pt.SendLine(testMsg)
	msg = pt.GetMessage()
	assert.Equal(t, "Foo", string(msg.Content))
}

func TestUTF16BEParserHandleMessages(t *testing.T) {
	pt := parsertesting.NewParserTester(New(UTF16BE))
	defer pt.Stop()
	testMsg := []byte{0x0, 'F', 0x0, 'o', 0x0, 'o'}
	pt.SendLine(testMsg)
	msg := pt.GetMessage()
	assert.Equal(t, "Foo", string(msg.Content))

	// We should support BOM
	testMsg = []byte{0xFE, 0xFF, 0x0, 'F', 0x0, 'o', 0x0, 'o'}
	pt.SendLine(testMsg)
	msg = pt.GetMessage()
	assert.Equal(t, "Foo", string(msg.Content))

	// BOM overrides endianness
	testMsg = []byte{0xFF, 0xFE, 'F', 0x0, 'o', 0x0, 'o', 0x0}
	pt.SendLine(testMsg)
	msg = pt.GetMessage()
	assert.Equal(t, "Foo", string(msg.Content))
}

func TestSHIFTJISParserHandleMessages(t *testing.T) {
	pt := parsertesting.NewParserTester(New(SHIFTJIS))
	defer pt.Stop()
	testMsg := []byte{0x93, 0xfa, 0x96, 0x7b}
	pt.SendLine(testMsg)
	msg := pt.GetMessage()
	assert.Equal(t, "日本", string(msg.Content))
}
