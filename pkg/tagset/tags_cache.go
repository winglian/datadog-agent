package tagset

// tagsCache caches Tags instances using purpose-specific cache keys.
//
// Note that tagsCache instances are not threadsafe
type tagsCache struct {
	// number of inserts between rotations (from constructor)
	insertsPerRotation int

	// inserts is the number of inserts performed on the current map.
	inserts int

	// searches is the number of searches performed on the current map.
	searches int

	// searchesHistory tracks the number of searches performed for each map,
	// during the time it was the current map.  This is maintained for
	// telemetry purposes.  The value at index 0 is not updated, but is
	// stored in the `searches` field.
	searchHistory []int

	// tagests contains the constitutent tagset maps. This is a slice of length
	// cacheCount.  The first map is the newest, into which new values will be
	// inserted.
	maps []map[uint64]*Tags
}

func newTagsCache(insertsPerRotation, cacheCount int) tagsCache {
	maps := make([]map[uint64]*Tags, cacheCount)
	for i := range maps {
		maps[i] = make(map[uint64]*Tags)
	}
	return tagsCache{
		insertsPerRotation: insertsPerRotation,
		inserts:            0,
		searches:           0,
		searchHistory:      make([]int, cacheCount),
		maps:               maps,
	}
}

// getCachedTags gets an element from the cache, calling miss() to generate the
// element if not found.
func (tc *tagsCache) getCachedTags(key uint64, miss func() *Tags) *Tags {
	v, ok := tc.search(key)
	if !ok {
		v = miss()
		tc.insert(key, v)
	}
	return v
}

// getCachedTagsErr is like getCachedTags, but works for miss() functions that can
// return an error.  Errors are not cached.
func (tc *tagsCache) getCachedTagsErr(key uint64, miss func() (*Tags, error)) (*Tags, error) {
	v, ok := tc.search(key)
	if !ok {
		var err error
		v, err = miss()
		if err != nil {
			return nil, err
		}
		tc.insert(key, v)
	}
	return v, nil
}

// search searches for a key in maps older than the first.  If found, the key
// is copied to the first map and returned.
func (tc *tagsCache) search(key uint64) (*Tags, bool) {
	tc.searches++
	v, ok := tc.maps[0][key]
	if ok {
		return v, true
	}

	cacheCount := len(tc.maps)
	for i := 1; i < cacheCount; i++ {
		v, ok = tc.maps[i][key]
		if ok {
			// "recache" this entry in the first map so that it's faster to
			// find next time
			tc.insert(key, v)
			return v, true
		}
	}

	return nil, false
}

// insert inserts a key into the first map.  It also performs rotation, if
// necessary.
func (tc *tagsCache) insert(key uint64, val *Tags) {
	tc.maps[0][key] = val
	tc.inserts++

	if tc.inserts < tc.insertsPerRotation {
		return
	}

	tc.rotate()
}

// rotate rotates the cache.
func (tc *tagsCache) rotate() {
	cacheCount := len(tc.maps)

	// try to allocate a new map of the size to which the last map
	// grew before being discarded.
	lastLen := len(tc.maps[cacheCount-1])

	tc.searchHistory[0] = tc.searches
	tc.searches = 0
	tc.inserts = 0
	copy(tc.maps[1:cacheCount], tc.maps[:cacheCount-1])
	copy(tc.searchHistory[1:cacheCount], tc.searchHistory[:cacheCount-1])

	// and initialize a new first map
	tc.maps[0] = make(map[uint64]*Tags, lastLen)
}

// telemetry retrieves the current telemetry for this cache
func (tc *tagsCache) telemetry() CacheTelemetry {
	tc.searchHistory[0] = tc.searches
	maps := make([]CacheMapTelemetry, len(tc.maps))
	for i := range maps {
		maps[i].Inserts = len(tc.maps[i])
		maps[i].Searches = tc.searchHistory[i]
	}
	return CacheTelemetry{Maps: maps}
}
