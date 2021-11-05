// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package metrics

import (
	"fmt"
	"math"

	"github.com/DataDog/datadog-agent/pkg/aggregator/ckey"
	"github.com/DataDog/datadog-agent/pkg/telemetry"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// ContextMetrics stores all the metrics by context key
type ContextMetrics map[ckey.ContextKey]Metric

// MakeContextMetrics returns a new ContextMetrics
func MakeContextMetrics() ContextMetrics {
	return ContextMetrics(make(map[ckey.ContextKey]Metric))
}

// AddSampleTelemetry counts number of new metrics added.
type AddSampleTelemetry struct {
	Total     telemetry.SimpleCounter
	Stateful  telemetry.SimpleCounter
	Stateless telemetry.SimpleCounter
}

// Inc should be called once for each new metric added to the map.
//
// isStateful should be the value returned by isStateful method for the new metric.
func (a *AddSampleTelemetry) Inc(isStateful bool) {
	a.Total.Inc()
	if isStateful {
		a.Stateful.Inc()
	} else {
		a.Stateless.Inc()
	}
}

// AddSample add a sample to the current ContextMetrics and initialize a new metrics if needed.
func (m ContextMetrics) AddSample(contextKey ckey.ContextKey, sample *MetricSample, timestamp float64, interval int64, t *AddSampleTelemetry) error {
	if math.IsInf(sample.Value, 0) || math.IsNaN(sample.Value) {
		return fmt.Errorf("sample with value '%v'", sample.Value)
	}
	if _, ok := m[contextKey]; !ok {
		switch sample.Mtype {
		case GaugeType:
			m[contextKey] = &Gauge{}
		case RateType:
			m[contextKey] = &Rate{}
		case CountType:
			m[contextKey] = &Count{}
		case MonotonicCountType:
			m[contextKey] = &MonotonicCount{}
		case HistogramType:
			m[contextKey] = NewHistogram(interval) // default histogram configuration (no call to `configure`) for now
		case HistorateType:
			m[contextKey] = NewHistorate(interval) // internal histogram has the configuration for now
		case SetType:
			m[contextKey] = NewSet()
		case CounterType:
			m[contextKey] = NewCounter(interval)
		default:
			err := fmt.Errorf("unknown sample metric type: %v", sample.Mtype)
			log.Error(err)
			return err
		}
		if t != nil {
			t.Inc(m[contextKey].isStateful())
		}
	}
	m[contextKey].addSample(sample, timestamp)
	return nil
}

// Flush flushes every metrics in the ContextMetrics.
// Returns the slice of Series and a map of errors by context key.
func (m ContextMetrics) Flush(timestamp float64) ([]*Serie, map[ckey.ContextKey]error) {
	var series []*Serie
	errors := make(map[ckey.ContextKey]error)

	for contextKey, metric := range m {
		metricSeries, err := metric.flush(timestamp)

		if err == nil {
			for _, serie := range metricSeries {
				serie.ContextKey = contextKey
				series = append(series, serie)
			}
		} else {
			switch err.(type) {
			case NoSerieError:
				// this error happens in nominal conditions and shouldn't be returned
			default:
				errors[contextKey] = err
			}
		}
	}

	return series, errors
}

// TimestampedContextMetrics TODO
type TimestampedContextMetrics struct {
	BucketTimestamp float64
	ContextMetrics  ContextMetrics
}

// TODO TODO
type TODO []TimestampedContextMetrics

// FlushAndClear FlushAndClear
func (t TODO) FlushAndClear(flush func([]*Serie)) map[ckey.ContextKey][]error {
	errors := make(map[ckey.ContextKey][]error)

	var lastcontextKey ckey.ContextKey
	var series []*Serie
	c := 0
	t.merge(func(contextKey ckey.ContextKey, m Metric, bucketTimestamp float64) {
		metricSeries, err := m.flush(bucketTimestamp)
		c += len(metricSeries)
		if err == nil { // $$ handle last series
			series = append(series, metricSeries...)
			if contextKey != lastcontextKey {
				for _, serie := range series {
					serie.ContextKey = contextKey
				}
				flush(series)
				lastcontextKey = contextKey
				series = series[:0]
			}

		} else {
			switch err.(type) {
			case NoSerieError:
				// this error happens in nominal conditions and shouldn't be returned
			default:
				errors[contextKey] = append(errors[contextKey], err)
			}
		}
	})
	fmt.Println("TEST42 flush", c)
	return errors
}

func (t *TODO) merge(callback func(ckey.ContextKey, Metric, float64)) {
	for i := 0; i < len(*t); i++ {
		for contextKey, metrics := range (*t)[i].ContextMetrics {
			callback(contextKey, metrics, (*t)[i].BucketTimestamp)

			for j := i + 1; j < len(*t); j++ {
				contextMetrics := (*t)[j].ContextMetrics
				if m, found := contextMetrics[contextKey]; found {
					callback(contextKey, m, (*t)[j].BucketTimestamp)
					delete(contextMetrics, contextKey)
				}
			}
		}
	}
}
