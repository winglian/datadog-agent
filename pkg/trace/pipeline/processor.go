// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// Package pipeline provides the entire pipeline of the agent in a single structure, allowing
// users to pass in a payload which is to be processed and written to the Datadog intake.
package pipeline

import (
	"errors"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/DataDog/datadog-agent/pkg/trace/config"
	"github.com/DataDog/datadog-agent/pkg/trace/config/features"
	"github.com/DataDog/datadog-agent/pkg/trace/event"
	"github.com/DataDog/datadog-agent/pkg/trace/filters"
	"github.com/DataDog/datadog-agent/pkg/trace/info"
	"github.com/DataDog/datadog-agent/pkg/trace/metrics/timing"
	"github.com/DataDog/datadog-agent/pkg/trace/obfuscate"
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/DataDog/datadog-agent/pkg/trace/sampler"
	"github.com/DataDog/datadog-agent/pkg/trace/stats"
	"github.com/DataDog/datadog-agent/pkg/trace/traceutil"
	"github.com/DataDog/datadog-agent/pkg/trace/writer"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// Payload specifies information about a set of traces received by the API.
type Payload struct {
	// Source specifies information about the source of these traces, such as:
	// language, interpreter, tracer version, etc.
	Source *info.TagStats

	// ContainerID specifies the container ID from where this payload originated, as
	// and if sent by the client.
	ContainerID string

	// ContainerTags specifies orchestrator tags corresponding to the origin of this
	// trace (e.g. K8S pod, Docker image, ECS, etc). They are of the type "k1:v1,k2:v2".
	ContainerTags string

	// Traces contains all the traces received in the payload
	Traces pb.Traces

	// ClientComputedTopLevel specifies that the client has already marked top-level
	// spans.
	ClientComputedTopLevel bool

	// ClientComputedStats reports whether the client has computed and sent over stats
	// so that the agent doesn't have to.
	ClientComputedStats bool

	// ClientDroppedP0s specifies the number of P0 traces chunks dropped by the client.
	ClientDroppedP0s int64
}

type Processor struct {
	In chan *Payload

	cfg *Config

	concentrator          *stats.Concentrator
	clientStatsAggregator *stats.ClientStatsAggregator
	blacklister           *filters.Blacklister
	replacer              *filters.Replacer
	prioritySampler       *sampler.PrioritySampler
	errorsSampler         *sampler.ErrorsSampler
	exceptionSampler      *sampler.ExceptionSampler
	noPrioritySampler     *sampler.NoPrioritySampler
	eventProcessor        *event.Processor
	obfuscator            *obfuscate.Obfuscator

	traceWriter *writer.TraceWriter
	statsWriter *writer.StatsWriter
}

type Config struct {
	Agent *config.AgentConfig
	Rates *sampler.DynamicConfig

	// PreSampleRate returns a pre-sample rate to use if ok is true. Off by default.
	PreSampleRate func() (rate float64, ok bool)
}

func NewProcessor(c *Config) *Processor {
	statsChan := make(chan pb.StatsPayload, 100)

	cfg := c.Agent
	if c.Rates == nil {
		c.Rates = sampler.NewDynamicConfig(cfg.DefaultEnv)
	}
	return &Processor{
		In:                    make(chan *Payload, 1000),
		cfg:                   c,
		concentrator:          stats.NewConcentrator(cfg, statsChan, time.Now()),
		clientStatsAggregator: stats.NewClientStatsAggregator(cfg, statsChan),
		blacklister:           filters.NewBlacklister(cfg.Ignore["resource"]),
		replacer:              filters.NewReplacer(cfg.ReplaceTags),
		prioritySampler:       sampler.NewPrioritySampler(cfg, c.Rates),
		errorsSampler:         sampler.NewErrorsSampler(cfg),
		exceptionSampler:      sampler.NewExceptionSampler(),
		noPrioritySampler:     sampler.NewNoPrioritySampler(cfg),
		eventProcessor:        newEventProcessor(cfg),
		traceWriter:           writer.NewTraceWriter(cfg),
		statsWriter:           writer.NewStatsWriter(cfg, statsChan),
		obfuscator:            obfuscate.NewObfuscator(cfg.Obfuscation),
	}
}

func (prc *Processor) Start() {
	for _, starter := range []interface{ Start() }{
		prc.concentrator,
		prc.clientStatsAggregator,
		prc.prioritySampler,
		prc.errorsSampler,
		prc.noPrioritySampler,
		prc.eventProcessor,
	} {
		starter.Start()
	}

	go prc.traceWriter.Run()
	go prc.statsWriter.Run()

	for i := 0; i < runtime.NumCPU(); i++ {
		go prc.work()
	}
}

func (prc *Processor) Stop() {
	for _, stopper := range []interface{ Stop() }{
		prc.concentrator,
		prc.clientStatsAggregator,
		prc.traceWriter,
		prc.statsWriter,
		prc.prioritySampler,
		prc.errorsSampler,
		prc.noPrioritySampler,
		prc.exceptionSampler,
		prc.eventProcessor,
		prc.obfuscator,
	} {
		stopper.Stop()
	}
}

func (p *Processor) work() {
	for {
		select {
		case pp, ok := <-p.In:
			if !ok {
				return
			}
			p.Process(pp)
		}
	}

}

const (
	// tagContainersTags specifies the name of the tag which holds key/value
	// pairs representing information about the container (Docker, EC2, etc).
	tagContainersTags = "_dd.tags.container"
)

// ProcessedTrace represents a trace being processed in the agent.
type ProcessedTrace struct {
	Trace            pb.Trace
	WeightedTrace    stats.WeightedTrace
	Root             *pb.Span
	Env              string
	ClientDroppedP0s bool
}

func (prc *Processor) Process(p *Payload) {
	if len(p.Traces) == 0 {
		log.Debugf("Skipping received empty payload")
		return
	}
	defer timing.Since("datadog.trace_agent.internal.process_payload_ms", time.Now())
	ts := p.Source
	ss := new(writer.SampledSpans)
	var envtraces []stats.EnvTrace
	prc.prioritySampler.CountClientDroppedP0s(p.ClientDroppedP0s)
	for _, t := range p.Traces {
		if len(t) == 0 {
			log.Debugf("Skipping received empty trace")
			continue
		}

		tracen := int64(len(t))
		atomic.AddInt64(&ts.SpansReceived, tracen)
		err := normalizeTrace(p.Source, t)
		if err != nil {
			log.Debugf("Dropping invalid trace: %s", err)
			atomic.AddInt64(&ts.SpansDropped, tracen)
			continue
		}

		// Root span is used to carry some trace-level metadata, such as sampling rate and priority.
		root := traceutil.GetRoot(t)

		if !prc.blacklister.Allows(root) {
			log.Debugf("Trace rejected by ignore resources rules. root: %v", root)
			atomic.AddInt64(&ts.TracesFiltered, 1)
			atomic.AddInt64(&ts.SpansFiltered, tracen)
			continue
		}

		if filteredByTags(root, prc.cfg.Agent.RequireTags, prc.cfg.Agent.RejectTags) {
			log.Debugf("Trace rejected as it fails to meet tag requirements. root: %v", root)
			atomic.AddInt64(&ts.TracesFiltered, 1)
			atomic.AddInt64(&ts.SpansFiltered, tracen)
			continue
		}

		// Extra sanitization steps of the trace.
		for _, span := range t {
			for k, v := range prc.cfg.Agent.GlobalTags {
				traceutil.SetMeta(span, k, v)
			}
			prc.obfuscator.Obfuscate(span)
			Truncate(span)
			if p.ClientComputedTopLevel {
				traceutil.UpdateTracerTopLevel(span)
			}
		}
		prc.replacer.Replace(t)

		{
			// this section sets up any necessary tags on the root:
			clientSampleRate := sampler.GetGlobalRate(root)
			sampler.SetClientRate(root, clientSampleRate)

			if rate, ok := prc.cfg.PreSampleRate(); ok {
				sampler.SetPreSampleRate(root, rate)
			}
			if p.ContainerTags != "" {
				traceutil.SetMeta(root, tagContainersTags, p.ContainerTags)
			}
		}
		if !p.ClientComputedTopLevel {
			// Figure out the top-level spans now as it involves modifying the Metrics map
			// which is not thread-safe while samplers and Concentrator might modify it too.
			traceutil.ComputeTopLevel(t)
		}

		env := prc.cfg.Agent.DefaultEnv
		if v := traceutil.GetEnv(t); v != "" {
			// this trace has a user defined env.
			env = v
		}
		pt := ProcessedTrace{
			Trace:            t,
			WeightedTrace:    stats.NewWeightedTrace(t, root),
			Root:             root,
			Env:              env,
			ClientDroppedP0s: p.ClientDroppedP0s > 0,
		}

		events, keep := prc.sample(ts, pt)
		if !p.ClientComputedStats {
			if envtraces == nil {
				envtraces = make([]stats.EnvTrace, 0, len(p.Traces))
			}
			envtraces = append(envtraces, stats.EnvTrace{
				Trace: pt.WeightedTrace,
				Env:   pt.Env,
			})
		}
		// TODO(piochelepiotr): Maybe we can skip some computation if stats are computed in the tracer and the trace is droped.
		if keep {
			ss.Traces = append(ss.Traces, traceutil.APITrace(t))
			ss.Size += t.Msgsize()
			ss.SpanCount += int64(len(t))
		}
		if len(events) > 0 {
			ss.Events = append(ss.Events, events...)
			ss.Size += pb.Trace(events).Msgsize()
		}
		if ss.Size > writer.MaxPayloadSize {
			prc.traceWriter.In <- ss
			ss = new(writer.SampledSpans)
		}
	}
	if ss.Size > 0 {
		prc.traceWriter.In <- ss
	}
	if len(envtraces) > 0 {
		in := stats.Input{Traces: envtraces}
		if !features.Has("disable_cid_stats") && prc.cfg.Agent.FargateOrchestrator != "" {
			// only allow the ContainerID stats dimension if we're in a Fargate instance
			// and it's not prohibited by the disable_cid_stats feature flag.
			in.ContainerID = p.ContainerID
		}
		prc.concentrator.In <- in
	}
}

// SetGlobalTagsUnsafe sets global tags to the agent configuration. Unsafe for concurrent use.
func (prc *Processor) SetGlobalTagsUnsafe(tags map[string]string) {
	prc.cfg.Agent.GlobalTags = tags
}

// sample decides whether the trace will be kept and extracts any APM events
// from it.
func (prc *Processor) sample(ts *info.TagStats, pt ProcessedTrace) (events []*pb.Span, keep bool) {
	priority, hasPriority := sampler.GetSamplingPriority(pt.Root)
	if hasPriority {
		ts.TracesPerSamplingPriority.CountSamplingPriority(priority)
	} else {
		atomic.AddInt64(&ts.TracesPriorityNone, 1)
	}

	if priority < 0 {
		return nil, false
	}

	sampled := prc.runSamplers(pt, hasPriority)
	events, numExtracted := prc.eventProcessor.Process(pt.Root, pt.Trace)

	atomic.AddInt64(&ts.EventsExtracted, int64(numExtracted))
	atomic.AddInt64(&ts.EventsSampled, int64(len(events)))

	return events, sampled
}

func newEventProcessor(conf *config.AgentConfig) *event.Processor {
	extractors := []event.Extractor{
		event.NewMetricBasedExtractor(),
	}
	if len(conf.AnalyzedSpansByService) > 0 {
		extractors = append(extractors, event.NewFixedRateExtractor(conf.AnalyzedSpansByService))
	} else if len(conf.AnalyzedRateByServiceLegacy) > 0 {
		extractors = append(extractors, event.NewLegacyExtractor(conf.AnalyzedRateByServiceLegacy))
	}
	return event.NewProcessor(extractors, conf.MaxEPS)
}

func filteredByTags(root *pb.Span, require, reject []*config.Tag) bool {
	for _, tag := range reject {
		if v, ok := root.Meta[tag.K]; ok && (tag.V == "" || v == tag.V) {
			return true
		}
	}
	for _, tag := range require {
		v, ok := root.Meta[tag.K]
		if !ok || (tag.V != "" && v != tag.V) {
			return true
		}
	}
	return false
}

// runSamplers runs all the agent's samplers on pt and returns the sampling decision
// along with the sampling rate.
func (prc *Processor) runSamplers(pt ProcessedTrace, hasPriority bool) bool {
	if hasPriority {
		return prc.samplePriorityTrace(pt)
	}
	return prc.sampleNoPriorityTrace(pt)
}

// samplePriorityTrace samples traces with priority set on them. PrioritySampler and
// ErrorSampler are run in parallel. The ExceptionSampler catches traces with rare top-level
// or measured spans that are not caught by PrioritySampler and ErrorSampler.
func (prc *Processor) samplePriorityTrace(pt ProcessedTrace) bool {
	if prc.prioritySampler.Sample(pt.Trace, pt.Root, pt.Env, pt.ClientDroppedP0s) {
		return true
	}
	if traceContainsError(pt.Trace) {
		return prc.errorsSampler.Sample(pt.Trace, pt.Root, pt.Env)
	}
	return prc.exceptionSampler.Sample(pt.Trace, pt.Root, pt.Env)
}

// sampleNoPriorityTrace samples traces with no priority set on them. The traces
// get sampled by either the score sampler or the error sampler if they have an error.
func (prc *Processor) sampleNoPriorityTrace(pt ProcessedTrace) bool {
	if traceContainsError(pt.Trace) {
		return prc.errorsSampler.Sample(pt.Trace, pt.Root, pt.Env)
	}
	return prc.noPrioritySampler.Sample(pt.Trace, pt.Root, pt.Env)
}

func traceContainsError(trace pb.Trace) bool {
	for _, span := range trace {
		if span.Error != 0 {
			return true
		}
	}
	return false
}

func (prc *Processor) FlushSync() error {
	if !prc.cfg.Agent.SynchronousFlushing {
		return errors.New("apm_conf.sync_flushing is not enabled. No data was sent to Datadog.")
	}
	if err := prc.statsWriter.FlushSync(); err != nil {
		return err
	}
	return prc.traceWriter.FlushSync()
}

func (prc *Processor) ProcessStats(in pb.ClientStatsPayload, lang, tracerVersion string) {
	prc.clientStatsAggregator.In <- prc.processStats(in, lang, tracerVersion)
}

func (prc *Processor) processStats(in pb.ClientStatsPayload, lang, tracerVersion string) pb.ClientStatsPayload {
	if features.Has("disable_cid_stats") || prc.cfg.Agent.FargateOrchestrator == "" {
		// this functionality is disabled by the disable_cid_stats feature flag
		// or we're not in a Fargate instance.
		in.ContainerID = ""
		in.Tags = nil
	}
	if in.Env == "" {
		in.Env = prc.cfg.Agent.DefaultEnv
	}
	in.Env = traceutil.NormalizeTag(in.Env)
	in.TracerVersion = tracerVersion
	in.Lang = lang
	for i, group := range in.Stats {
		n := 0
		for _, b := range group.Stats {
			normalizeStatsGroup(&b, lang)
			if !prc.blacklister.AllowsStat(&b) {
				continue
			}
			prc.obfuscator.ObfuscateStatsGroup(&b)
			prc.replacer.ReplaceStatsGroup(&b)
			group.Stats[n] = b
			n++
		}
		in.Stats[i].Stats = group.Stats[:n]
		mergeDuplicates(in.Stats[i])
	}
	return in
}

func mergeDuplicates(s pb.ClientStatsBucket) {
	indexes := make(map[stats.Aggregation]int, len(s.Stats))
	for i, g := range s.Stats {
		a := stats.NewAggregationFromGroup(g)
		if j, ok := indexes[a]; ok {
			s.Stats[j].Hits += g.Hits
			s.Stats[j].Errors += g.Errors
			s.Stats[j].Duration += g.Duration
			s.Stats[i].Hits = 0
			s.Stats[i].Errors = 0
			s.Stats[i].Duration = 0
		} else {
			indexes[a] = i
		}
	}
}
