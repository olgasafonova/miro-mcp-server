package miro

import (
	"sync"
	"time"
)

// =============================================================================
// Cache Configuration
// =============================================================================

// CacheConfig holds cache configuration options.
type CacheConfig struct {
	// BoardTTL is the TTL for board-level data (list boards, board details).
	BoardTTL time.Duration

	// ItemTTL is the TTL for item-level data (get item, list items).
	ItemTTL time.Duration

	// TagTTL is the TTL for tag data.
	TagTTL time.Duration

	// MaxEntries is the maximum number of cache entries (0 = unlimited).
	MaxEntries int
}

// DefaultCacheConfig returns sensible default cache configuration.
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		BoardTTL:   2 * time.Minute,
		ItemTTL:    1 * time.Minute,
		TagTTL:     2 * time.Minute,
		MaxEntries: 1000,
	}
}

// =============================================================================
// Cache
// =============================================================================

// Cache provides a thread-safe cache with TTL and LRU eviction.
type Cache struct {
	mu         sync.RWMutex
	entries    map[string]*CacheEntry
	accessList []string // For LRU eviction
	config     CacheConfig
	stats      CacheStats
}

// CacheEntry holds cached data with metadata.
type CacheEntry struct {
	Data       interface{}
	ExpiresAt  time.Time
	TTL        time.Duration
	AccessedAt time.Time
}

// CacheStats tracks cache performance metrics.
type CacheStats struct {
	Hits       int64
	Misses     int64
	Evictions  int64
	Invalidations int64
}

// NewCache creates a new cache with the given configuration.
func NewCache(config CacheConfig) *Cache {
	return &Cache{
		entries:    make(map[string]*CacheEntry),
		accessList: make([]string, 0),
		config:     config,
	}
}

// Get retrieves a cached value if valid.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		c.mu.Lock()
		c.stats.Misses++
		c.mu.Unlock()
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.stats.Misses++
		c.mu.Unlock()
		return nil, false
	}

	c.mu.Lock()
	entry.AccessedAt = time.Now()
	c.stats.Hits++
	c.mu.Unlock()

	return entry.Data, true
}

// Set stores a value in the cache with the specified TTL.
func (c *Cache) Set(key string, data interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest entries if at capacity
	if c.config.MaxEntries > 0 && len(c.entries) >= c.config.MaxEntries {
		c.evictOldest()
	}

	now := time.Now()
	c.entries[key] = &CacheEntry{
		Data:       data,
		ExpiresAt:  now.Add(ttl),
		TTL:        ttl,
		AccessedAt: now,
	}
	c.accessList = append(c.accessList, key)
}

// Invalidate removes a specific key from the cache.
func (c *Cache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.entries[key]; ok {
		delete(c.entries, key)
		c.stats.Invalidations++
	}
}

// InvalidatePrefix removes all keys with the given prefix.
func (c *Cache) InvalidatePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.entries {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.entries, key)
			c.stats.Invalidations++
		}
	}
}

// InvalidateBoard invalidates all cache entries for a specific board.
func (c *Cache) InvalidateBoard(boardID string) {
	c.InvalidatePrefix("board:" + boardID)
	c.InvalidatePrefix("items:" + boardID)
	c.InvalidatePrefix("item:" + boardID)
	c.InvalidatePrefix("tags:" + boardID)
	c.InvalidatePrefix("connectors:" + boardID)
}

// InvalidateItem invalidates cache entries for a specific item.
func (c *Cache) InvalidateItem(boardID, itemID string) {
	c.Invalidate("item:" + boardID + ":" + itemID)
	// Also invalidate items list since it may have changed
	c.InvalidatePrefix("items:" + boardID)
}

// Clear removes all entries from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
	c.accessList = make([]string, 0)
}

// Stats returns cache statistics.
func (c *Cache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// Size returns the number of entries in the cache.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// evictOldest removes the oldest entries to make room (must be called with lock held).
func (c *Cache) evictOldest() {
	// Find entries to evict: expired first, then oldest accessed
	now := time.Now()
	toEvict := make([]string, 0)

	// First pass: find expired entries
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			toEvict = append(toEvict, key)
		}
	}

	// If not enough expired, evict oldest accessed
	if len(toEvict) == 0 && len(c.accessList) > 0 {
		// Evict 10% of entries or at least 1
		evictCount := len(c.entries) / 10
		if evictCount < 1 {
			evictCount = 1
		}

		// Find oldest entries from access list
		for _, key := range c.accessList {
			if _, ok := c.entries[key]; ok {
				toEvict = append(toEvict, key)
				if len(toEvict) >= evictCount {
					break
				}
			}
		}
	}

	// Evict
	for _, key := range toEvict {
		delete(c.entries, key)
		c.stats.Evictions++
	}

	// Rebuild access list
	newAccessList := make([]string, 0, len(c.entries))
	for _, key := range c.accessList {
		if _, ok := c.entries[key]; ok {
			newAccessList = append(newAccessList, key)
		}
	}
	c.accessList = newAccessList
}

// =============================================================================
// Cache Key Builders
// =============================================================================

// CacheKeyBoard returns the cache key for a board.
func CacheKeyBoard(boardID string) string {
	return "board:" + boardID
}

// CacheKeyBoards returns the cache key for the boards list.
func CacheKeyBoards(query string) string {
	if query == "" {
		return "boards:all"
	}
	return "boards:query:" + query
}

// CacheKeyItem returns the cache key for an item.
func CacheKeyItem(boardID, itemID string) string {
	return "item:" + boardID + ":" + itemID
}

// CacheKeyItems returns the cache key for items list.
func CacheKeyItems(boardID, itemType, cursor string) string {
	return "items:" + boardID + ":" + itemType + ":" + cursor
}

// CacheKeyTags returns the cache key for tags list.
func CacheKeyTags(boardID string) string {
	return "tags:" + boardID
}

// CacheKeyConnectors returns the cache key for connectors list.
func CacheKeyConnectors(boardID string) string {
	return "connectors:" + boardID
}

// CacheKeyUserInfo returns the cache key for user info.
func CacheKeyUserInfo() string {
	return "token:userinfo"
}
