package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Entry represents a cached response with expiration
type Entry struct {
	Value     interface{}
	ExpiresAt time.Time
}

// Cache is a thread-safe LRU cache with TTL
type Cache struct {
	enabled    bool
	maxEntries int
	ttl        time.Duration
	entries    map[string]*Entry
	lru        []string // LRU order, most recent at end
	mu         sync.RWMutex
}

// New creates a new cache
func New(enabled bool, maxEntries int, ttl time.Duration) *Cache {
	return &Cache{
		enabled:    enabled,
		maxEntries: maxEntries,
		ttl:        ttl,
		entries:    make(map[string]*Entry),
		lru:        make([]string, 0, maxEntries),
	}
}

// Get retrieves a value from the cache if it exists and hasn't expired
func (c *Cache) Get(key string) (interface{}, bool) {
	if !c.enabled {
		return nil, false
	}

	c.mu.RLock()
	entry, exists := c.entries[key]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.removeLRU(key)
		c.mu.Unlock()
		return nil, false
	}

	// Update LRU on hit
	c.mu.Lock()
	c.touchLRU(key)
	c.mu.Unlock()

	return entry.Value, true
}

// Set stores a value in the cache
func (c *Cache) Set(key string, value interface{}) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict if at capacity
	if len(c.entries) >= c.maxEntries {
		if len(c.lru) > 0 {
			oldest := c.lru[0]
			delete(c.entries, oldest)
			c.lru = c.lru[1:]
		}
	}

	c.entries[key] = &Entry{
		Value:     value,
		ExpiresAt: time.Now().Add(c.ttl),
	}
	c.touchLRU(key)
}

// HashRequest creates a cache key from a request
func HashRequest(request interface{}) string {
	data, _ := json.Marshal(request)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// touchLRU moves key to the end (most recent)
func (c *Cache) touchLRU(key string) {
	// Remove existing
	c.removeLRU(key)
	// Add to end
	c.lru = append(c.lru, key)
}

// removeLRU removes key from LRU list
func (c *Cache) removeLRU(key string) {
	for i, k := range c.lru {
		if k == key {
			c.lru = append(c.lru[:i], c.lru[i+1:]...)
			return
		}
	}
}
