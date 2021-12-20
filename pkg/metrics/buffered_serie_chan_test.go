// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// +build test

package metrics

import (
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

func TestBufferedSerieChan(t *testing.T) {
	c := NewBufferedSerieChan(100, 2)
	stop := make(chan struct{})
	go func() {
		for n := 0; n < 100; n++ {
			c.Put(&Serie{Name: strconv.Itoa(n)}, stop)
		}
		c.Close()
	}()

	serie, ok := c.Get()
	r := require.New(t)

	n := 0
	for ; ok; serie, ok = c.Get() {
		r.Equal(strconv.Itoa(n), serie.Name)
		n++
	}
	r.Equal(100, n)
}

func BenchmarkBufferedSerieChan(b *testing.B) {
	stop := make(chan struct{})
	for n := 1000; n < 10000; n = (n * 5) / 4 {
		b.Run("BufferedSerieChan"+strconv.Itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			c := NewBufferedSerieChan(100, n)

			go func() {
				for n := 0; n < b.N; n++ {
					c.Put(nil, stop)
				}
				c.Close()
			}()
			_, ok := c.Get()
			for ; ok; _, ok = c.Get() {
			}
		})
	}
}

func BenchmarkSerieChan(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	c := make(chan int, 100*100)

	go func() {
		for n := 0; n < b.N; n++ {
			c <- n
		}
		close(c)
	}()
	_, ok := <-c
	for ; ok; _, ok = <-c {
	}
}
