package cache

import (
	"sync"
	"time"
)

// SecretCache stores last known good secret values in memory.
// Thread-safe for concurrent access.
// Secrets automatically expire after maxAge.
// No disk persistence - cleared on pod restart.
type SecretCache struct {
	mu      sync.RWMutex
	secrets map[string]*CachedSecret
	maxAge  time.Duration
}

// CachedSecret represents a secret value with timestamp.
type CachedSecret struct {
	Data      []byte
	FetchedAt time.Time
}

// NewSecretCache creates a new in-memory secret cache.
// maxAge: maximum age before secrets are considered stale (0 = 24 hours default).
func NewSecretCache(maxAge time.Duration) *SecretCache {
	if maxAge == 0 {
		maxAge = 24 * time.Hour
	}

	return &SecretCache{
		secrets: make(map[string]*CachedSecret),
		maxAge:  maxAge,
	}
}

// Get retrieves a cached secret if it exists and is not expired.
// Returns (secret, true) if found and valid, (nil, false) otherwise.
//
// Thread-safe for concurrent reads.
func (c *SecretCache) Get(name string) (*CachedSecret, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.secrets[name]
	if !ok {
		return nil, false
	}

	age := time.Since(cached.FetchedAt)
	if age > c.maxAge {
		return nil, false  // Expired
	}

	return cached, true
}

// Set stores a secret in the cache with current timestamp.
// Thread-safe for concurrent writes.
func (c *SecretCache) Set(name string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.secrets[name] = &CachedSecret{
		Data:      data,
		FetchedAt: time.Now(),
	}
}

// Age returns how long ago a secret was fetched.
// Returns 0 if secret not in cache.
func (c *SecretCache) Age(name string) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if cached, ok := c.secrets[name]; ok {
		return time.Since(cached.FetchedAt)
	}
	return 0
}

// Size returns the number of secrets in the cache.
func (c *SecretCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.secrets)
}

// Clear removes all secrets from the cache.
func (c *SecretCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.secrets = make(map[string]*CachedSecret)
}
