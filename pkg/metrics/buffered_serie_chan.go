// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package metrics

import (
	"sync"
)

// BufferedSerieChan TODO
type BufferedSerieChan struct {
	serieChan chan []*Serie
	pool      *sync.Pool
	putSlice  []*Serie
	getSlice  []*Serie
	getIndex  int
}

// NewBufferedSerieChan TODO
func NewBufferedSerieChan(chanSize int, bufferSize int) *BufferedSerieChan {
	pool := &sync.Pool{
		New: func() interface{} {
			return make([]*Serie, 0, bufferSize)
		},
	}
	return &BufferedSerieChan{
		serieChan: make(chan []*Serie, chanSize),
		pool:      pool,
		putSlice:  pool.Get().([]*Serie),
	}
}

// Put TODO
func (c *BufferedSerieChan) Put(serie *Serie, stop chan struct{}) {
	if cap(c.putSlice) <= len(c.putSlice) {
		select {
		case c.serieChan <- c.putSlice:
		case <-stop:
		}
		c.putSlice = c.pool.Get().([]*Serie)[:0]
	}
	c.putSlice = append(c.putSlice, serie)
}

// Close TODO
func (c *BufferedSerieChan) Close() {
	if len(c.putSlice) > 0 { // $$$ needed?
		c.serieChan <- c.putSlice
	}
	close(c.serieChan)
}

// Get TODO
func (c *BufferedSerieChan) Get() (*Serie, bool) {
	if c.getIndex >= len(c.getSlice) {
		c.getIndex = 0
		if c.getSlice != nil {
			c.pool.Put(c.getSlice)
		}

		var ok bool
		if c.getSlice, ok = <-c.serieChan; !ok {
			return nil, false
		}
	}
	value := c.getSlice[c.getIndex]
	c.getIndex++
	return value, true
}
