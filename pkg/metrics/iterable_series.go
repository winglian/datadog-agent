// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package metrics

import (
	"errors"
	"sync/atomic"

	jsoniter "github.com/json-iterator/go"
)

// IterableSeries represents an iterable collection of Serie.
// Serie can be appended to IterableSeries while IterableSeries is serialized
type IterableSeries struct {
	c                   *BufferedSerieChan
	receiverStoppedChan chan struct{}
	callback            func(*Serie)
	current             *Serie
	count               uint64
}

// NewIterableSeries creates a new instance of *IterableSeries
// `callback` is called each time `Append` is called.
// `chanSize` is the internal channel buffer size
func NewIterableSeries(callback func(*Serie), chanSize int, bufferSize int) *IterableSeries {
	return &IterableSeries{
		c:                   NewBufferedSerieChan(chanSize, bufferSize),
		receiverStoppedChan: make(chan struct{}),
		callback:            callback,
		current:             nil,
	}
}

// Append appends a serie
func (series *IterableSeries) Append(serie *Serie) {
	series.callback(serie)
	atomic.AddUint64(&series.count, 1)
	series.c.Put(serie, series.receiverStoppedChan)
}

// SeriesCount returns the number of series appended with `IterableSeries.Append`.
func (series *IterableSeries) SeriesCount() uint64 {
	return atomic.LoadUint64(&series.count)
}

// SenderStopped must be called when sender stop calling Append.
func (series *IterableSeries) SenderStopped() {
	series.c.Close()
}

// IterationStopped must be called when the receiver stops calling `MoveNext`.
// This function prevents the case when the receiver stops iterating before the
// end of the iteration because of an error and so blocks the sender forever
// as no goroutine read the channel.
func (series *IterableSeries) IterationStopped() {
	close(series.receiverStoppedChan)
}

// WriteHeader writes the payload header for this type
func (series *IterableSeries) WriteHeader(stream *jsoniter.Stream) error {
	return writeHeader(stream)
}

// WriteFooter writes the payload footer for this type
func (series *IterableSeries) WriteFooter(stream *jsoniter.Stream) error {
	return writeFooter(stream)
}

// WriteCurrentItem writes the json representation of an item
func (series *IterableSeries) WriteCurrentItem(stream *jsoniter.Stream) error {
	if series.Current() == nil {
		return errors.New("nil serie")
	}
	return writeItem(stream, series.Current())
}

// DescribeCurrentItem returns a text description for logs
func (series *IterableSeries) DescribeCurrentItem() string {
	if series.Current() == nil {
		return "nil serie"
	}
	return describeItem(series.Current())
}

// MoveNext advances to the next element.
// Returns false for the end of the iteration.
func (series *IterableSeries) MoveNext() bool {
	var ok bool
	series.current, ok = series.c.Get()
	return ok
}

// Current returns the current serie.
func (series *IterableSeries) Current() *Serie {
	return series.current
}
