package proxychain

import "time"

// Cache provides an interface for caching mechanisms.
// It supports operations to get, set, and invalidate cache entries.
// Implementations should ensure thread safety, efficiency
type Cache interface {
	// Get Retrieves a cached value by its key. Returns the value and a boolean indicating
	Get(key string) (value interface{}, found bool)

	// Set - Stores a value associated with a key in the cache for a specified time-to-live (ttl).
	// If ttl is zero, the cache item has no expiration.
	Set(key string, value interface{}, ttl time.Duration)

	// Invalidate - Removes a value from the cache by its key. If the key does not exist,
	// it should perform a no-op or return a suitable error.
	Invalidate(key string) error
}
