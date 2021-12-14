// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package metrics

import "encoding/json"

// CompositeTags TODO
type CompositeTags struct {
	tags1 []string
	tags2 []string
}

// NewCompositeTags TODO
func NewCompositeTags(tags1 []string, tags2 []string) *CompositeTags {
	return &CompositeTags{
		tags1: tags1,
		tags2: tags2,
	}
}

// ToSliceString TODO
func (t *CompositeTags) ToSliceString() []string {
	return append(t.tags1, t.tags2...)
}

// ForEachErr TODO
func (t *CompositeTags) ForEachErr(callback func(tag string) error) error {
	for _, t := range t.tags1 {
		if err := callback(t); err != nil {
			return err
		}
	}
	for _, t := range t.tags2 {
		if err := callback(t); err != nil {
			return err
		}
	}
	return nil
}

// ForEach TODO
func (t *CompositeTags) ForEach(callback func(tag string)) {
	for _, t := range t.tags1 {
		callback(t)
	}
	for _, t := range t.tags2 {
		callback(t)
	}
}

// Find TODO
func (t *CompositeTags) Find(callback func(tag string) bool) bool {
	for _, t := range t.tags1 {
		if callback(t) {
			return true
		}
	}
	for _, t := range t.tags2 {
		if callback(t) {
			return true
		}
	}
	return false
}

// Len TODO
func (t *CompositeTags) Len() int {
	return len(t.tags1) + len(t.tags2)
}

// Append TODO
func (t *CompositeTags) Append(tags []string) *CompositeTags {
	return NewCompositeTags(append(t.tags1, tags...), t.tags2)
}

// MarshalJSON TODO
func (t *CompositeTags) MarshalJSON() ([]byte, error) {
	return json.Marshal(append(t.tags1, t.tags2...))
}

// UnmarshalJSON TODO
func (t *CompositeTags) UnmarshalJSON(b []byte) error {
	t.tags2 = nil
	return json.Unmarshal(b, &t.tags1)
}
