// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package writer

import (
	"compress/gzip"
	"io"
	"math"
	"strings"
	"sync/atomic"
	"time"

	"github.com/DataDog/datadog-agent/pkg/trace/config"
	"github.com/DataDog/datadog-agent/pkg/trace/info"
	"github.com/DataDog/datadog-agent/pkg/trace/logutil"
	"github.com/DataDog/datadog-agent/pkg/trace/metrics"
	"github.com/DataDog/datadog-agent/pkg/trace/metrics/timing"
	"github.com/DataDog/datadog-agent/pkg/trace/pipeline"
	"github.com/DataDog/datadog-agent/pkg/util/log"

	"github.com/tinylib/msgp/msgp"
)

const (
	// pathPipelineStats is the target host API path for delivering pipeline stats.
	pathPipelineStats = "/api/v0.1/pipeline_stats"
	defaultPipelineConnectionLimit = 20
	// approximation of number of bytes per pipeline stats entry
	bytesPerPipelineStatsEntry = 100
	// maxPipelineStatsPayloadSize is the maximum size of pipeline stats payloads.
	// Datadog intake API limits a compressed payload to ~3MB, but
	// let's have the default ensure we don't have paylods > 1.5 MB to have some spare room. Since
	// some entries could be larger than 100 bytes
	maxPipelineStatsPayloadSize = 1500000
	maxEntriesPerPipelineStatsPayload = maxPipelineStatsPayloadSize / bytesPerPipelineStatsEntry
	defaultMaxMemory = 250*1024*1024
)

// PipelineStatsWriter ingests stats buckets and flushes them to the API.
type PipelineStatsWriter struct {
	In      chan pipeline.ClientStatsPayload
	senders []*sender
	stop    chan struct{}
	stats   *info.PipelineStatsWriterInfo
	easylog *logutil.ThrottledLogger
}

// NewPipelineStatsWriter returns a new StatsWriter. It must be started using Run.
func NewPipelineStatsWriter(cfg *config.AgentConfig) *PipelineStatsWriter {
	sw := &PipelineStatsWriter{
		In: make(chan pipeline.ClientStatsPayload, 100),
		stats:     &info.PipelineStatsWriterInfo{},
		stop:      make(chan struct{}),
		easylog:   logutil.NewThrottled(5, 10*time.Second), // no more than 5 messages every 10 seconds
	}
	climit := cfg.PipelineStatsWriter.ConnectionLimit
	if climit == 0 {
		climit = defaultPipelineConnectionLimit
	}
	qsize := cfg.PipelineStatsWriter.QueueSize
	if qsize == 0 {
		// default to 25% of maximum memory.
		maxmem := cfg.MaxMemory / 4
		if maxmem == 0 {
			maxmem = defaultMaxMemory
		}
		qsize = int(math.Max(1, maxmem/maxPipelineStatsPayloadSize))
	}
	log.Debugf("Pipeline stats writer initialized (climit=%d qsize=%d)", climit, qsize)
	sw.senders = newSenders(cfg, sw, pathPipelineStats, climit, qsize)
	return sw
}

func (w *PipelineStatsWriter) handlePayload(statsPayload pipeline.ClientStatsPayload) {
	entries := 0
	for _, bucket := range statsPayload.Stats {
		entries += len(bucket.Stats)
	}
	if entries < maxEntriesPerPayload {
		return
	}
}

// Run starts the StatsWriter, making it ready to receive stats and report metrics.
func (w *PipelineStatsWriter) Run() {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	defer close(w.stop)
	for {
		select {
		case stats := <-w.In:
			w.SendPayload(pipeline.StatsPayload{Stats: []pipeline.ClientStatsPayload{stats}})
		case <-t.C:
			w.report()
		case <-w.stop:
			return
		}
	}
}

// Stop stops a running StatsWriter.
func (w *PipelineStatsWriter) Stop() {
	w.stop <- struct{}{}
	<-w.stop
	stopSenders(w.senders)
}

// SendPayload sends a stats payload to the Datadog backend.
func (w *PipelineStatsWriter) SendPayload(p pipeline.StatsPayload) {
	req := newPayload(map[string]string{
		headerLanguages:    strings.Join(info.Languages(), "|"),
		"Content-Type":     "application/msgpack",
		"Content-Encoding": "gzip",
	})
	if err := encodePipelinePayload(req.body, p); err != nil {
		log.Errorf("Pipeline stats encoding error: %v", err)
		return
	}
	sendPayloads(w.senders, req, false)
}

// encodePipelinePayload encodes the payload as Gzipped msgPack into w.
func encodePipelinePayload(w io.Writer, payload pipeline.StatsPayload) error {
	gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
	if err != nil {
		return err
	}
	defer func() {
		if err := gz.Close(); err != nil {
			log.Errorf("Error closing gzip stream when writing stats payload: %v", err)
		}
	}()
	return msgp.Encode(gz, &payload)
}

// todo[piochelepiotr] What is this line doing?
var _ eventRecorder = (*PipelineStatsWriter)(nil)

func (w *PipelineStatsWriter) report() {
	metrics.Count("datadog.trace_agent.pipeline_stats_writer.client_payloads", atomic.SwapInt64(&w.stats.ClientPayloads, 0), nil, 1)
	metrics.Count("datadog.trace_agent.pipeline_stats_writer.payloads", atomic.SwapInt64(&w.stats.Payloads, 0), nil, 1)
	metrics.Count("datadog.trace_agent.pipeline_stats_writer.stats_buckets", atomic.SwapInt64(&w.stats.StatsBuckets, 0), nil, 1)
	metrics.Count("datadog.trace_agent.pipeline_stats_writer.stats_entries", atomic.SwapInt64(&w.stats.StatsEntries, 0), nil, 1)
	metrics.Count("datadog.trace_agent.pipeline_stats_writer.bytes", atomic.SwapInt64(&w.stats.Bytes, 0), nil, 1)
	metrics.Count("datadog.trace_agent.pipeline_stats_writer.retries", atomic.SwapInt64(&w.stats.Retries, 0), nil, 1)
	metrics.Count("datadog.trace_agent.pipeline_stats_writer.splits", atomic.SwapInt64(&w.stats.Splits, 0), nil, 1)
	metrics.Count("datadog.trace_agent.pipeline_stats_writer.errors", atomic.SwapInt64(&w.stats.Errors, 0), nil, 1)
}

// recordEvent implements eventRecorder.
func (w *PipelineStatsWriter) recordEvent(t eventType, data *eventData) {
	if data != nil {
		metrics.Histogram("datadog.trace_agent.pipeline_stats_writer.connection_fill", data.connectionFill, nil, 1)
		metrics.Histogram("datadog.trace_agent.pipeline_stats_writer.queue_fill", data.queueFill, nil, 1)
	}
	switch t {
	case eventTypeRetry:
		log.Debugf("Retrying to flush pipeline stats payload (error: %q)", data.err)
		atomic.AddInt64(&w.stats.Retries, 1)

	case eventTypeSent:
		log.Debugf("Flushed pipeline stats to the API; time: %s, bytes: %d", data.duration, data.bytes)
		timing.Since("datadog.trace_agent.pipeline_stats_writer.flush_duration", time.Now().Add(-data.duration))
		atomic.AddInt64(&w.stats.Bytes, int64(data.bytes))
		atomic.AddInt64(&w.stats.Payloads, 1)

	case eventTypeRejected:
		log.Warnf("Pipeline stats writer payload rejected by edge: %v", data.err)
		atomic.AddInt64(&w.stats.Errors, 1)

	case eventTypeDropped:
		w.easylog.Warn("Pipeline stats writer queue full. Payload dropped (%.2fKB).", float64(data.bytes)/1024)
		metrics.Count("datadog.trace_agent.pipeline_stats_writer.dropped", 1, nil, 1)
		metrics.Count("datadog.trace_agent.pipeline_stats_writer.dropped_bytes", int64(data.bytes), nil, 1)
	}
}
