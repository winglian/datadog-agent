// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021-present Datadog, Inc.

package tags

import (
	"fmt"
	"math/bits"

	"github.com/DataDog/datadog-agent/pkg/tagset"
	"github.com/DataDog/datadog-agent/pkg/telemetry"
)

// TODO: move ../tagset_tlm.go in here??
// TODO: use an interface so "disabled" is just empty functions
// TODO: keep existing config??

// activeTagset is used to keep track of tag slices shared by the contexts.
type activeTagset struct {
	// tagCount is the number of tags in this tagset
	tagCount int
	refs     uint64
}

// Tlm tracks telemetry about the aggregator's use of tags.
type Tlm struct {
	// tagsByKey stores the active tagsets, keyed by their hash
	tagsByKey map[uint64]*activeTagset

	// cap is a shortcut to the current capacity of tagsByKey
	cap int

	// enabled is true if tagset telemetry is enabled
	enabled bool

	// telemetry stores the actual telemetry for the tagset
	telemetry storeTelemetry
}

// NewTlm returns new empty Store.
func NewTlm(enabled bool, name string) *Tlm {
	return &Tlm{
		tagsByKey: map[uint64]*activeTagset{},
		enabled:   enabled,
		telemetry: newStoreTelemetry(name),
	}
}

// Use registers a use of the given tags instance in the aggregator.  This
// must be balanced by a later Release.
func (tc *Tlm) Use(tags *tagset.Tags) {
	if !tc.enabled {
		return
	}

	key := tags.Hash()
	entry := tc.tagsByKey[key]
	if entry != nil {
		entry.refs++
		tc.telemetry.hits.Inc()
	} else {
		tc.tagsByKey[key] = &activeTagset{
			tagCount: tags.Len(),
			refs:     1,
		}
		tc.cap++
		tc.telemetry.miss.Inc()
	}

	return
}

// Release decrements internal reference counter, potentially marking
// the entry as unused.
func (tc *Tlm) Release(tags *tagset.Tags) {
	if e, ok := tc.tagsByKey[tags.Hash()]; ok {
		e.refs--
	}
}

// Shrink will try to release memory if cache usage drops low enough.
func (tc *Tlm) Shrink() {
	stats := entryStats{}
	for key, entry := range tc.tagsByKey {
		if entry.refs == 0 {
			delete(tc.tagsByKey, key)
		} else {
			stats.visit(entry)
		}
	}

	if len(tc.tagsByKey) < tc.cap/2 {
		new := make(map[uint64]*activeTagset, len(tc.tagsByKey))
		for k, v := range tc.tagsByKey {
			new[k] = v
		}
		tc.cap = len(new)
		tc.tagsByKey = new
	}

	tc.updateTelemetry(&stats)
}

func (tc *Tlm) updateTelemetry(s *entryStats) {
	t := &tc.telemetry

	tlmMaxEntries.Set(float64(tc.cap), t.name)
	tlmEntries.Set(float64(len(tc.tagsByKey)), t.name)

	for i := 0; i < 3; i++ {
		tlmTagsetRefsCnt.Set(float64(s.refsFreq[i]), t.name, fmt.Sprintf("%d", i+1))
	}
	for i := 3; i < 8; i++ {
		tlmTagsetRefsCnt.Set(float64(s.refsFreq[i]), t.name, fmt.Sprintf("%d", 1<<(i-1)))
	}

	tlmTagsetMinTags.Set(float64(s.minSize), t.name)
	tlmTagsetMaxTags.Set(float64(s.maxSize), t.name)
	tlmTagsetSumTags.Set(float64(s.sumSize), t.name)
}

func newCounter(name string, help string, tags ...string) telemetry.Counter {
	return telemetry.NewCounter("aggregator_tags_store", name,
		append([]string{"cache_instance_name"}, tags...), help)
}

func newGauge(name string, help string, tags ...string) telemetry.Gauge {
	return telemetry.NewGauge("aggregator_tags_store", name,
		append([]string{"cache_instance_name"}, tags...), help)
}

var (
	tlmHits          = newCounter("hits_total", "number of times cache already contained the tags")
	tlmMiss          = newCounter("miss_total", "number of times cache did not contain the tags")
	tlmEntries       = newGauge("entries", "number of entries in the tags cache")
	tlmMaxEntries    = newGauge("max_entries", "maximum number of entries since last shrink")
	tlmTagsetMinTags = newGauge("tagset_min_tags", "minimum number of tags in a tagset")
	tlmTagsetMaxTags = newGauge("tagset_max_tags", "maximum number of tags in a tagset")
	tlmTagsetSumTags = newGauge("tagset_sum_tags", "total number of tags stored in all tagsets by the cache")
	tlmTagsetRefsCnt = newGauge("tagset_refs_count", "distribution of usage count of tagsets in the cache", "ge")
)

type storeTelemetry struct {
	hits telemetry.SimpleCounter
	miss telemetry.SimpleCounter
	name string
}

func newStoreTelemetry(name string) storeTelemetry {
	return storeTelemetry{
		hits: tlmHits.WithValues(name),
		miss: tlmMiss.WithValues(name),
		name: name,
	}
}

type entryStats struct {
	refsFreq [8]uint64
	minSize  int
	maxSize  int
	sumSize  int
	count    int
}

func (s *entryStats) visit(e *activeTagset) {
	r := e.refs
	if r < 4 {
		s.refsFreq[r-1]++
	} else if r < 64 {
		s.refsFreq[bits.Len64(r)]++ // Len(4) = 3, Len(63) = 6
	} else {
		s.refsFreq[7]++
	}

	n := e.tagCount
	if n < s.minSize || s.count == 0 {
		s.minSize = n
	}
	if n > s.maxSize {
		s.maxSize = n
	}
	s.sumSize += n
	s.count++
}
