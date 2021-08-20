// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-present Datadog, Inc.

//+build zlib

package stream

import (
	"bytes"
	"compress/zlib"
	"sync"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/forwarder"
	"github.com/DataDog/datadog-agent/pkg/serializer/marshaler"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/gogo/protobuf/proto"
)

// PBPayloadBuilder is used to build payloads. PBPayloadBuilder allocates memory based
// on what was previously need to serialize payloads. Keep that in mind and
// use multiple PBPayloadBuilders for different sources.
type PBPayloadBuilder struct {
	maxPayloadSize      int
	maxUncompressedSize int
	outputSizeHint      int
	shareAndLockBuffers bool
	slicer              pbSlicer
	input, output       *bytes.Buffer
	mu                  sync.Mutex
}

func NewPBPayloadBuilder() *PBPayloadBuilder {
	maxPayloadSize := config.Datadog.GetInt("serializer_max_payload_size")
	maxUncompressedSize := config.Datadog.GetInt("serializer_max_uncompressed_payload_size")

	return &PBPayloadBuilder{
		maxPayloadSize:      maxPayloadSize,
		maxUncompressedSize: maxUncompressedSize,
		outputSizeHint:      4096,
		shareAndLockBuffers: false,
	}
}

// Build serializes a metadata payload and sends it to the forwarder
func (b *PBPayloadBuilder) Build(m marshaler.StreamPBMarshaler) (forwarder.Payloads, error) {
	return b.BuildWithOnErrItemTooBigPolicy(m, DropItemOnErrItemTooBig)
}

// BuildWithOnErrItemTooBigPolicy serializes a metadata payload and sends it to the forwarder
func (b *PBPayloadBuilder) BuildWithOnErrItemTooBigPolicy(m marshaler.StreamPBMarshaler, policy OnErrItemTooBigPolicy) (forwarder.Payloads, error) {
	output := bytes.NewBuffer(make([]byte, 0, b.outputSizeHint))

	var payloads forwarder.Payloads
	var i int
	itemCount := m.Len()

	// take a safety margin off the expected payload sizes
	targetPayloadSize := 15 * b.maxPayloadSize / 16
	targetUncompressedSize := 15 * b.maxUncompressedSize / 16

	zipper := zlib.NewWriter(output)

	for i < itemCount {
		sliceSize := b.slicer.estimateItemCountFor(targetUncompressedSize, targetPayloadSize)
		if sliceSize < 0 {
			sliceSize = 1
		}
	Remarshal:
		end := i + sliceSize
		if end > itemCount {
			end = itemCount
		}
		item, err := proto.Marshal(m.SliceMessage(i, end))
		if err != nil {
			log.Warnf("error marshalling at item %d, skipping: %s", i, err)
			i++
			continue
		}

		uncompressedSize := len(item)

		_, err = zipper.Write(item)
		if err != nil {
			return nil, err
		}
		zipper.Flush()
		compressedSize := output.Len()

		b.slicer.recordSlice(sliceSize, uncompressedSize, compressedSize)

		// if this payload is too big, try again, ensuring that we strictly reduce the
		// size of the slice
		if compressedSize > b.maxPayloadSize || uncompressedSize > b.maxUncompressedSize {
			if sliceSize == 1 {
				if policy == FailOnErrItemTooBig {
					return nil, ErrItemTooBig
				} else {
					i++ // TODO: warn about skipping
				}
				continue
			} else if sliceSize < 4 {
				sliceSize--
				goto Remarshal
			} else {
				sliceSize = sliceSize * 3 / 4
				goto Remarshal
			}
		}

		// steal the output buffer
		b.outputSizeHint = output.Cap()
		payload := output.Bytes()
		payloads = append(payloads, &payload)
		output = bytes.NewBuffer(make([]byte, 0, b.outputSizeHint))

		// advance our pointer into the sliced items
		i = end
	}

	return payloads, nil
}

// A pbSliceHistory records the results of encoding a slice, in service of
// predicting future slice sizes.
type pbSliceHistory struct {
	itemCount        int
	uncompressedSize int
	compressedSize   int
}

type pbSlicer struct {
	history []pbSliceHistory
}

// recordSlice records the measurement of a payload
func (slicer *pbSlicer) recordSlice(itemCount, uncompressedSize, compressedSize int) {
	slicer.history = append(slicer.history, pbSliceHistory{itemCount, uncompressedSize, compressedSize})
	if len(slicer.history) > 10 {
		slicer.history = slicer.history[len(slicer.history)-10:]
	}
}

// estimateItemCountFor returns an estimate of the itemCount that will produce
// a payload with at most the given uncompressed or compressed sizes.
func (slicer *pbSlicer) estimateItemCountFor(uncompressedSize, compressedSize int) int {
	var sumItems, sumUncomp, sumComp int

	// take a guess if we have no history
	if len(slicer.history) == 0 {
		return 100
	}

	for _, h := range slicer.history {
		sumItems += h.itemCount
		sumUncomp += h.uncompressedSize
		sumComp += h.compressedSize
	}

	uncompPerItem := float64(sumUncomp) / float64(sumItems)
	compPerItem := float64(sumComp) / float64(sumItems)

	uncompItems := float64(uncompressedSize) / uncompPerItem
	compItems := float64(compressedSize) / compPerItem

	if uncompItems < compItems {
		return int(uncompItems)
	} else {
		return int(compItems)
	}
}
