// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.Datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package util

import (
	"sort"

	"github.com/twmb/murmur3"
)

// Tag is a string with a hash attached.
type Tag struct {
	Data *string
	Hash uint64
}

// Text returns the tag string.
func (t *Tag) Text() string {
	return *t.Data
}

// NewTags creates new tags slice from a list of strings.
func NewTags(tags ...string) []Tag {
	if len(tags) == 0 {
		return nil
	}

	ts := make([]Tag, 0, len(tags))
	for _, t := range tags {
		tptr := t
		ts = append(ts, Tag{
			Data: &tptr,
			Hash: murmur3.StringSum64(t),
		})
	}
	return ts
}

// SortTags sorts tags slice inplace.
func SortTags(tags []Tag) {
	sort.Slice(tags, func(i, j int) bool { return *tags[i].Data < *tags[j].Data })
}

// TagsBuilder allows to build a slice of tags to generate the context while
// reusing the same internal slice.
type TagsBuilder struct {
	data []string
	hash []uint64
}

// NewTagsBuilder returns a new empty TagsBuilder.
func NewTagsBuilder() *TagsBuilder {
	return &TagsBuilder{
		// Slice will grow as more tags are added to it. 128 tags
		// should be enough for most metrics.
		data: make([]string, 0, 128),
		hash: make([]uint64, 0, 128),
	}
}

// NewTagsBuilderFromSlice return a new TagsBuilder with the input slice for
// it's internal buffer.
func NewTagsBuilderFromSlice(tags []string) *TagsBuilder {
	hash := make([]uint64, 0, len(tags))
	for _, t := range tags {
		hash = append(hash, murmur3.StringSum64(t))
	}
	return &TagsBuilder{
		data: tags,
		hash: hash,
	}
}

// NewTagsBuilderFromTags creates new TagsBuilder from a tags slice.
func NewTagsBuilderFromTags(tags []Tag) *TagsBuilder {
	tb := NewTagsBuilder()
	tb.AppendTags(tags)
	return tb
}

// Append appends tags to the builder
func (tb *TagsBuilder) Append(tags ...string) {
	for _, t := range tags {
		tb.data = append(tb.data, t)
		tb.hash = append(tb.hash, murmur3.StringSum64(t))
	}
}

// AppendTags appends tags and their hashes to the builder
func (tb *TagsBuilder) AppendTags(tags []Tag) {
	for _, t := range tags {
		tb.data = append(tb.data, *t.Data)
		tb.hash = append(tb.hash, t.Hash)
	}
}

// AppendToTags appends contents of tb to a slice of tags.
func (tb *TagsBuilder) AppendToTags(tags []Tag) []Tag {
	for i := range tb.data {
		tags = append(tags, Tag{
			Data: &tb.data[i],
			Hash: tb.hash[i],
		})
	}
	return tags
}

// AppendBuilder appends tags from src, re-using hashes.
func (tb *TagsBuilder) AppendBuilder(src *TagsBuilder) {
	tb.data = append(tb.data, src.data...)
	tb.hash = append(tb.hash, src.hash...)
}

// SortUniq sorts and remove duplicate in place
func (tb *TagsBuilder) SortUniq() {
	if tb.Len() < 2 {
		return
	}

	sort.Sort(tb)

	j := 0
	for i := 1; i < len(tb.data); i++ {
		if tb.hash[i] == tb.hash[j] && tb.data[i] == tb.data[j] {
			continue
		}
		j++
		tb.data[j] = tb.data[i]
		tb.hash[j] = tb.hash[i]
	}

	tb.Truncate(j + 1)
}

// Reset resets the size of the builder to 0 without discaring the internal
// buffer
func (tb *TagsBuilder) Reset() {
	// we keep the internal buffer but reset size
	tb.data = tb.data[0:0]
	tb.hash = tb.hash[0:0]
}

// Truncate retains first n tags in the buffer without discarding the internal buffer
func (tb *TagsBuilder) Truncate(len int) {
	tb.data = tb.data[0:len]
	tb.hash = tb.hash[0:len]
}

// Get returns the internal slice
func (tb *TagsBuilder) Get() []string {
	if tb == nil {
		return nil
	}
	return tb.data
}

// Hashes returns the internal slice of tag hashes
func (tb *TagsBuilder) Hashes() []uint64 {
	if tb == nil {
		return nil
	}
	return tb.hash
}

// Copy makes a copy of the internal slice
func (tb *TagsBuilder) Copy() []string {
	if tb == nil {
		return nil
	}
	return append(make([]string, 0, len(tb.data)), tb.data...)
}

// Less implements sort.Interface.Less
func (tb *TagsBuilder) Less(i, j int) bool {
	// FIXME(vickenty): could sort using hashes, which is faster, but a lot of tests check for order.
	return tb.data[i] < tb.data[j]
}

// Slice returns a shared slice of tb's internal data.
func (tb *TagsBuilder) Slice(i, j int) *TagsBuilder {
	return &TagsBuilder{
		data: tb.data[i:j],
		hash: tb.hash[i:j],
	}
}

// Swap implements sort.Interface.Swap
func (tb *TagsBuilder) Swap(i, j int) {
	tb.hash[i], tb.hash[j] = tb.hash[j], tb.hash[i]
	tb.data[i], tb.data[j] = tb.data[j], tb.data[i]
}

// Len implements sort.Interface.Len
func (tb *TagsBuilder) Len() int {
	if tb == nil {
		return 0
	}
	return len(tb.data)
}
