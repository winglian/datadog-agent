// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux_bpf
// +build linux_bpf

package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTXPath(t *testing.T) {
	t.Run("regular path", func(t *testing.T) {
		cHTTP := &cHTTP{
			request_fragment: requestFragment(
				[]byte("GET /foo/bar?var1=value HTTP/1.1\nHost: example.com\nUser-Agent: example-browser/1.0"),
			),
		}

		tx := newTX(cHTTP)
		assert.Equal(t, "/foo/bar", string(tx.path))
	})

	t.Run("null termination", func(t *testing.T) {
		cHTTP := &cHTTP{
			request_fragment: requestFragment(
				// This probably isn't a valid HTTP request
				// (since it's missing a version before the end),
				// but if the null byte isn't handled
				// then the path becomes "/foo/\x00bar"
				[]byte("GET /foo/\x00bar?var1=value HTTP/1.1\nHost: example.com\nUser-Agent: example-browser/1.0"),
			),
		}

		tx := newTX(cHTTP)
		assert.Equal(t, "/foo/", string(tx.path))
	})
}

func TestLatency(t *testing.T) {
	tx := httpTX{
		responseLastSeen: 2e6,
		requestStarted:   1e6,
	}
	// quantization brings it down
	assert.Equal(t, 999424.0, tx.RequestLatency())
}

func requestFragment(fragment []byte) [HTTPBufferSize]_Ctype_char {
	var b [HTTPBufferSize]_Ctype_char
	for i := 0; i < len(b) && i < len(fragment); i++ {
		b[i] = _Ctype_char(fragment[i])
	}
	return b
}
