package tagset

// Telemetry represents the telemetry information available for a caching factory.
//
// A caching factory contains a number of purpose-specific caches, each defining
// its uint64 key in a different way.  For each such cache, it contains one or more
// maps that are rotated periodically.
type Telemetry struct {
	// FactoryName is the name of the factory this telemetry represents (as
	// given to NewCachingFactoryWithTelemetry)
	FactoryName string

	// caches contains the CacheTelemetry for each cache, indexed by the cache name.
	Caches map[string]CacheTelemetry
}

// CacheTelemetry represents the telemetry information for a specific cache in a
// caching factory.
type CacheTelemetry struct {
	// maps contains the CacheMapTelemetry for each map in the cache.  The first
	// map, at index 0, is the map into which items are currently being inserted.
	Maps []CacheMapTelemetry
}

// CacheMapTelemetry represents telemetry information for a specific map in a
// cache.
type CacheMapTelemetry struct {
	// Inserts gives the number of tagsets in this map.  This includes cache
	// misses that were fetched from an older map as well as cache misses
	// that resulted in generation of a new Tags instance.
	Inserts int

	// Searches gives the number of searches in this map, during the time it
	// was the current map.  The ratio of Searches to Inserts gives the cache
	// hit rate.
	Searches int
}
