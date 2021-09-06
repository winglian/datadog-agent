package dogstatsd

import (
	"math/bits"

	"github.com/twmb/murmur3"

	"github.com/DataDog/datadog-agent/pkg/telemetry"
	telemetry_utils "github.com/DataDog/datadog-agent/pkg/telemetry/utils"
	"github.com/DataDog/datadog-agent/pkg/util"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

var (
	// Amount of resets of the string interner used in dogstatsd
	// Note that it's not ideal because there is many allocated string interner
	// (one per worker) but it'll still give us an insight (and it's comparable
	// as long as the amount of worker is stable).
	tlmSIResets = telemetry.NewCounter("dogstatsd", "string_interner_resets",
		nil, "Amount of resets of the string interner used in dogstatsd")
)

type stringInternerEntry struct {
	hash uint64
	data string
}

// stringInterner is a string cache providing a longer life for strings,
// helping to avoid GC runs because they're re-used many times instead of
// created every time.
type stringInterner struct {
	entries []stringInternerEntry
	hashes  []uint16
	mask    uint64
	used    int
	maxSize int
	// telemetry
	tlmEnabled bool
}

func newStringInterner(maxSize int) *stringInterner {
	if maxSize <= 0 {
		maxSize = 500
	}
	size := 1 << bits.Len(uint(maxSize+maxSize/8))
	return &stringInterner{
		entries:    make([]stringInternerEntry, size),
		hashes:     make([]uint16, size),
		mask:       uint64(size - 1),
		maxSize:    maxSize,
		tlmEnabled: telemetry_utils.IsEnabled(),
	}
}

// LoadOrStore is the string-only version of LoadOrStoreTag.
func (i *stringInterner) LoadOrStore(key []byte) string {
	return *i.LoadOrStoreTag(key).Data
}

// LoadOrStoreTag always returns the string from the cache, adding it into the
// cache if needed.
// If we need to store a new entry and the cache is at its maximum capacity,
// it is reset.
func (i *stringInterner) LoadOrStoreTag(key []byte) util.Tag {
	h := murmur3.Sum64(key)
	pos := h & i.mask
	hh := 0x8000 | (uint16(h) >> 1)
	beg := pos
	var e *stringInternerEntry

	for {
		if i.hashes[pos] == 0 {
			if i.used >= i.maxSize {
				log.Debug("clearing the string interner cache")
				if i.tlmEnabled {
					tlmSIResets.Inc()
				}
				i.hashes = make([]uint16, i.mask+1)
				i.used = 0
				pos = h & i.mask
			}
			i.hashes[pos] = hh
			e = &i.entries[pos]
			e.hash = h
			e.data = string(key)
			i.used++
			break
		}
		if i.hashes[pos] == hh {
			e = &i.entries[pos]
			if e.hash == h && e.data == string(key) {
				break
			}
		}
		pos = (pos + 1) & i.mask
		if pos == beg {
			panic("interner wrapped around, insufficient capacity")
		}
	}

	return util.Tag{
		Data: &e.data,
		Hash: e.hash,
	}
}
