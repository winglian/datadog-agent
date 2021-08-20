// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package marshaler

import (
	"bytes"

	"github.com/gogo/protobuf/proto"
	jsoniter "github.com/json-iterator/go"
)

// Marshaler is an interface for metrics that are able to serialize themselves to JSON and protobuf
type Marshaler interface {
	MarshalJSON() ([]byte, error)
	Marshal() ([]byte, error)
	SplitPayload(int) ([]Marshaler, error)
	MarshalSplitCompress(*BufferContext) ([]*[]byte, error)
}

// StreamPBMarshaler is an interface for metrics that are able to serialize themselves to protobuf
// in a streaming fashion.  This interface indexes "items" in the value by an integer index, and
// will only serialize each item once.
type StreamPBMarshaler interface {
	Marshaler

	// SliceMessage returns a message containing the sliced items.  The stream marhsaller will
	// be "smart" about slice sizes, targeting a full payload, but may still call this function
	// several times for overlapping slices.  Implementations should, where possible, use caching
	// to make such repeated calls efficient.
	SliceMessage(start, end int) proto.Message

	// Len returns the number of items the marshaler will produce
	Len() int

	// DescribeItem describes an item for use in debugging and errormessages
	DescribeItem(i int) string
}

// StreamJSONMarshaler is an interface for metrics that are able to serialize themselves in a stream
type StreamJSONMarshaler interface {
	Marshaler
	WriteHeader(*jsoniter.Stream) error
	WriteFooter(*jsoniter.Stream) error
	WriteItem(*jsoniter.Stream, int) error
	Len() int
	DescribeItem(i int) string
}

// BufferContext contains the buffers used for MarshalSplitCompress so they can be shared between invocations
type BufferContext struct {
	CompressorInput   *bytes.Buffer
	CompressorOutput  *bytes.Buffer
	PrecompressionBuf []byte
}

// DefaultBufferContext initialize the default compression buffers
func DefaultBufferContext() *BufferContext {
	return &BufferContext{
		bytes.NewBuffer(make([]byte, 0, 1024)),
		bytes.NewBuffer(make([]byte, 0, 1024)),
		make([]byte, 1024),
	}
}
