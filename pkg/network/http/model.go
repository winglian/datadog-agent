// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux_bpf
// +build linux_bpf

package http

import (
	"unsafe"
)

/*
#include "../ebpf/c/http-types.h"
*/
import "C"

const HTTPBufferSize = int(C.HTTP_BUFFER_SIZE)

// for testing purposes only
type cHTTP = C.http_transaction_t

type httpTX struct {
	tup        C.conn_tuple_t
	statusCode int
	path       []byte
	method     Method

	requestStarted   int64
	responseLastSeen int64

	tags uint64
}

func newTX(http *C.http_transaction_t) httpTX {
	// TODO: pool either httpTX objects or byte slices for the path
	return httpTX{
		tup:              http.tup,
		statusCode:       int(http.response_status_code),
		requestStarted:   int64(http.request_started),
		responseLastSeen: int64(http.response_last_seen),
		method:           Method(http.request_method),
		tags:             uint64(http.tags),
		path:             extractPath(nil, unsafe.Pointer(&http.request_fragment), HTTPBufferSize),
	}
}

// StatusClass returns an integer representing the status code class
// Example: a 404 would return 400
func (tx *httpTX) StatusClass() int {
	return (int(tx.statusCode) / 100) * 100
}

// RequestLatency returns the latency of the request in nanoseconds
func (tx *httpTX) RequestLatency() float64 {
	return nsTimestampToFloat(uint64(tx.responseLastSeen - tx.requestStarted))
}

// Incomplete returns true if the transaction contains only the request or response information
// This happens in the context of localhost with NAT, in which case we join the two parts in userspace
func (tx *httpTX) Incomplete() bool {
	return tx.requestStarted == 0 || tx.statusCode == 0
}

// Tags returns an uint64 representing the tags bitfields
// Tags are defined here : pkg/network/ebpf/kprobe_types.go
func (tx *httpTX) Tags() uint64 {
	return tx.tags
}

// below is copied from pkg/trace/stats/statsraw.go
// 10 bits precision (any value will be +/- 1/1024)
const roundMask uint64 = 1 << 10

// nsTimestampToFloat converts a nanosec timestamp into a float nanosecond timestamp truncated to a fixed precision
func nsTimestampToFloat(ns uint64) float64 {
	var shift uint
	for ns > roundMask {
		ns = ns >> 1
		shift++
	}
	return float64(ns << shift)
}

func extractPath(buffer []byte, p unsafe.Pointer, limit int) []byte {
	buffer = buffer[:0]

	var (
		i int
		b byte
	)
	for i = 0; i < limit; i++ {
		b = *(*byte)(unsafe.Pointer(uintptr(p) + uintptr(i)))

		if b == ' ' {
			break
		}

		if b == 0 {
			return buffer
		}
	}

	i++
	for j := i; j < limit; j++ {
		b = *(*byte)(unsafe.Pointer(uintptr(p) + uintptr(j)))

		if b == 0 || b == '?' || b == ' ' {
			return buffer
		}

		buffer = append(buffer, b)
	}

	return buffer
}
