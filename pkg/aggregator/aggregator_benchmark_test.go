// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package aggregator

import (
	// stdlib

	"net/http"
	"strconv"
	"testing"
	"time"

	// 3p

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/forwarder"
	"github.com/DataDog/datadog-agent/pkg/metrics"
	"github.com/DataDog/datadog-agent/pkg/serializer"
)

// ForwarderMock a mocked forwarder to be use in other module to test their dependencies with the forwarder
type ForwarderMock struct {
	serieLength int
	serieCount  int
}

// Start updates the internal mock struct
func (tf *ForwarderMock) Start() error {
	return nil
}

// Stop updates the internal mock struct
func (tf *ForwarderMock) Stop() {
}

// SubmitV1Series updates the internal mock struct
func (tf *ForwarderMock) SubmitV1Series(payload forwarder.Payloads, extra http.Header) error {
	for _, p := range payload {
		tf.serieLength += len(*p)
	}
	tf.serieCount += len(payload)
	return nil
}

// SubmitV1Intake updates the internal mock struct
func (tf *ForwarderMock) SubmitV1Intake(payload forwarder.Payloads, extra http.Header) error {
	return nil
}

// SubmitV1CheckRuns updates the internal mock struct
func (tf *ForwarderMock) SubmitV1CheckRuns(payload forwarder.Payloads, extra http.Header) error {
	return nil
}

// SubmitEvents updates the internal mock struct
func (tf *ForwarderMock) SubmitEvents(payload forwarder.Payloads, extra http.Header) error {
	return nil
}

// SubmitSeries updates the internal mock struct
func (tf *ForwarderMock) SubmitSeries(payload forwarder.Payloads, extra http.Header) error {
	return nil
}

// SubmitServiceChecks updates the internal mock struct
func (tf *ForwarderMock) SubmitServiceChecks(payload forwarder.Payloads, extra http.Header) error {
	return nil
}

// SubmitSketchSeries updates the internal mock struct
func (tf *ForwarderMock) SubmitSketchSeries(payload forwarder.Payloads, extra http.Header) error {
	return nil
}

// SubmitHostMetadata updates the internal mock struct
func (tf *ForwarderMock) SubmitHostMetadata(payload forwarder.Payloads, extra http.Header) error {
	return nil
}

// SubmitAgentChecksMetadata updates the internal mock struct
func (tf *ForwarderMock) SubmitAgentChecksMetadata(payload forwarder.Payloads, extra http.Header) error {
	return nil
}

// SubmitMetadata updates the internal mock struct
func (tf *ForwarderMock) SubmitMetadata(payload forwarder.Payloads, extra http.Header) error {
	return nil
}

// SubmitProcessChecks mock
func (tf *ForwarderMock) SubmitProcessChecks(payload forwarder.Payloads, extra http.Header) (chan forwarder.Response, error) {
	return nil, nil
}

// SubmitProcessDiscoveryChecks mock
func (tf *ForwarderMock) SubmitProcessDiscoveryChecks(payload forwarder.Payloads, extra http.Header) (chan forwarder.Response, error) {
	return nil, nil
}

// SubmitRTProcessChecks mock
func (tf *ForwarderMock) SubmitRTProcessChecks(payload forwarder.Payloads, extra http.Header) (chan forwarder.Response, error) {
	return nil, nil
}

// SubmitContainerChecks mock
func (tf *ForwarderMock) SubmitContainerChecks(payload forwarder.Payloads, extra http.Header) (chan forwarder.Response, error) {
	return nil, nil
}

// SubmitRTContainerChecks mock
func (tf *ForwarderMock) SubmitRTContainerChecks(payload forwarder.Payloads, extra http.Header) (chan forwarder.Response, error) {
	return nil, nil
}

// SubmitConnectionChecks mock
func (tf *ForwarderMock) SubmitConnectionChecks(payload forwarder.Payloads, extra http.Header) (chan forwarder.Response, error) {
	return nil, nil
}

// SubmitOrchestratorChecks mock
func (tf *ForwarderMock) SubmitOrchestratorChecks(payload forwarder.Payloads, extra http.Header, payloadType int) (chan forwarder.Response, error) {
	return nil, nil
}

type AggregatorData struct {
	agg  *BufferedAggregator
	mock *ForwarderMock
}

func createAggregator(enable bool) *AggregatorData {
	// setup the aggregator
	mock := &ForwarderMock{}
	s := serializer.NewSerializer(mock, nil)
	config.Datadog.Set("aggregator_flush_metrics_and_serialize_in_parallel", enable)
	defer config.Datadog.Set("aggregator_flush_metrics_and_serialize_in_parallel", nil)
	return &AggregatorData{
		agg:  NewBufferedAggregator(s, nil, "hostname", DefaultFlushInterval),
		mock: mock,
	}
}

func BenchmarkTimeSamplerFlushPerf(b *testing.B) {

	var aggregators []*AggregatorData
	for n := 0; n < b.N; n++ {
		agg := createAggregator(true)
		for n := 0; n < 500*1000; n++ {
			sample := metrics.MetricSample{
				Name:       "my.metric.name",
				Value:      1,
				Mtype:      metrics.CounterType,
				Tags:       []string{"foo", "bar", strconv.Itoa(n)},
				SampleRate: 1,
				Timestamp:  20,
			}
			agg.agg.addSample(&sample, 20)
		}
		aggregators = append(aggregators, agg)
	}

	b.ReportAllocs()
	b.ResetTimer()
	// heapSys := uint64(0)
	// heapSysMax := uint64(0)
	for n := 0; n < b.N; n++ {
		// var m runtime.MemStats
		// runtime.ReadMemStats(&m)

		aggregators[n].agg.flushSeriesAndSketches(time.Now(), true)
		// l := aggregators[n].mock.serieLength

		// var newMemStats runtime.MemStats
		// runtime.ReadMemStats(&newMemStats)
		// d := newMemStats.HeapSys - m.HeapSys
		// heapSys += d
		// if d > heapSysMax {
		// 	heapSysMax = d
		// }
	}

	// b.ReportMetric(float64(heapSys/uint64(b.N)), "heapSys")
	// b.ReportMetric(float64(heapSysMax), "heapSysMax")
}

// BenchmarkTimeSamplerFlushPerf-8               10         929974179 ns/op        217820041 B/op   2000994 allocs/op
// BenchmarkTimeSamplerFlushPerf-8               10         942701025 ns/op        217817533 B/op   2000994 allocs/op

// BenchmarkTimeSamplerFlushPerf-8   	       2	 945464583 ns/op	217824404 B/op	 2001001 allocs/op

// BenchmarkTimeSamplerFlushPerf-8               10         753030565 ns/op        233883731 B/op   2001479 allocs/op
// BenchmarkTimeSamplerFlushPerf-8               10        1094612739 ns/op        244777174 B/op   2000524 allocs/op
