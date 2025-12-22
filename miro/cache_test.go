package miro

import (
	"testing"
	"time"
)

func TestCache_GetSet(t *testing.T) {
	cache := NewCache(DefaultCacheConfig())

	// Test set and get
	cache.Set("key1", "value1", time.Minute)
	val, ok := cache.Get("key1")
	if !ok {
		t.Fatal("expected to find key1")
	}
	if val != "value1" {
		t.Errorf("expected value1, got %v", val)
	}

	// Test cache miss
	_, ok = cache.Get("nonexistent")
	if ok {
		t.Error("expected cache miss for nonexistent key")
	}
}

func TestCache_Expiration(t *testing.T) {
	cache := NewCache(DefaultCacheConfig())

	// Set with very short TTL
	cache.Set("expiring", "value", 1*time.Millisecond)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Should be expired
	_, ok := cache.Get("expiring")
	if ok {
		t.Error("expected cache miss for expired key")
	}
}

func TestCache_Invalidate(t *testing.T) {
	cache := NewCache(DefaultCacheConfig())

	cache.Set("key1", "value1", time.Minute)
	cache.Set("key2", "value2", time.Minute)

	// Invalidate single key
	cache.Invalidate("key1")
	_, ok := cache.Get("key1")
	if ok {
		t.Error("expected key1 to be invalidated")
	}

	// key2 should still exist
	_, ok = cache.Get("key2")
	if !ok {
		t.Error("expected key2 to still exist")
	}
}

func TestCache_InvalidatePrefix(t *testing.T) {
	cache := NewCache(DefaultCacheConfig())

	cache.Set("items:board1:type1", "value1", time.Minute)
	cache.Set("items:board1:type2", "value2", time.Minute)
	cache.Set("items:board2:type1", "value3", time.Minute)
	cache.Set("other:key", "value4", time.Minute)

	// Invalidate all items for board1
	cache.InvalidatePrefix("items:board1")

	_, ok := cache.Get("items:board1:type1")
	if ok {
		t.Error("expected items:board1:type1 to be invalidated")
	}

	_, ok = cache.Get("items:board1:type2")
	if ok {
		t.Error("expected items:board1:type2 to be invalidated")
	}

	// board2 items should still exist
	_, ok = cache.Get("items:board2:type1")
	if !ok {
		t.Error("expected items:board2:type1 to still exist")
	}

	// other keys should still exist
	_, ok = cache.Get("other:key")
	if !ok {
		t.Error("expected other:key to still exist")
	}
}

func TestCache_InvalidateBoard(t *testing.T) {
	cache := NewCache(DefaultCacheConfig())

	// Set various cache entries for a board
	cache.Set("board:board1", "board data", time.Minute)
	cache.Set("items:board1:sticky_note:", "items data", time.Minute)
	cache.Set("item:board1:item123", "item data", time.Minute)
	cache.Set("tags:board1", "tags data", time.Minute)
	cache.Set("connectors:board1", "connectors data", time.Minute)
	cache.Set("board:board2", "other board", time.Minute)

	// Invalidate board1
	cache.InvalidateBoard("board1")

	// All board1 entries should be gone
	_, ok := cache.Get("board:board1")
	if ok {
		t.Error("expected board:board1 to be invalidated")
	}

	_, ok = cache.Get("items:board1:sticky_note:")
	if ok {
		t.Error("expected items:board1 to be invalidated")
	}

	_, ok = cache.Get("item:board1:item123")
	if ok {
		t.Error("expected item:board1:item123 to be invalidated")
	}

	_, ok = cache.Get("tags:board1")
	if ok {
		t.Error("expected tags:board1 to be invalidated")
	}

	_, ok = cache.Get("connectors:board1")
	if ok {
		t.Error("expected connectors:board1 to be invalidated")
	}

	// board2 should still exist
	_, ok = cache.Get("board:board2")
	if !ok {
		t.Error("expected board:board2 to still exist")
	}
}

func TestCache_InvalidateItem(t *testing.T) {
	cache := NewCache(DefaultCacheConfig())

	cache.Set("item:board1:item123", "item data", time.Minute)
	cache.Set("items:board1:sticky_note:", "list data", time.Minute)
	cache.Set("item:board1:item456", "other item", time.Minute)

	// Invalidate specific item
	cache.InvalidateItem("board1", "item123")

	_, ok := cache.Get("item:board1:item123")
	if ok {
		t.Error("expected item:board1:item123 to be invalidated")
	}

	// Items list should also be invalidated
	_, ok = cache.Get("items:board1:sticky_note:")
	if ok {
		t.Error("expected items list to be invalidated")
	}

	// Other item should still exist
	_, ok = cache.Get("item:board1:item456")
	if !ok {
		t.Error("expected item:board1:item456 to still exist")
	}
}

func TestCache_Clear(t *testing.T) {
	cache := NewCache(DefaultCacheConfig())

	cache.Set("key1", "value1", time.Minute)
	cache.Set("key2", "value2", time.Minute)

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("expected cache size 0, got %d", cache.Size())
	}
}

func TestCache_Stats(t *testing.T) {
	cache := NewCache(DefaultCacheConfig())

	// Set a value
	cache.Set("key1", "value1", time.Minute)

	// Hit
	cache.Get("key1")

	// Miss
	cache.Get("nonexistent")

	stats := cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("expected 1 hit, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", stats.Misses)
	}
}

func TestCache_Eviction(t *testing.T) {
	config := CacheConfig{
		BoardTTL:   time.Minute,
		ItemTTL:    time.Minute,
		TagTTL:     time.Minute,
		MaxEntries: 5,
	}
	cache := NewCache(config)

	// Fill cache beyond capacity
	for i := 0; i < 10; i++ {
		cache.Set("key"+string(rune('0'+i)), "value", time.Minute)
	}

	// Cache size should not exceed max
	if cache.Size() > 5 {
		t.Errorf("expected cache size <= 5, got %d", cache.Size())
	}
}

func TestCacheKeyBuilders(t *testing.T) {
	tests := []struct {
		fn       func() string
		expected string
	}{
		{func() string { return CacheKeyBoard("board123") }, "board:board123"},
		{func() string { return CacheKeyBoards("") }, "boards:all"},
		{func() string { return CacheKeyBoards("test") }, "boards:query:test"},
		{func() string { return CacheKeyItem("board1", "item1") }, "item:board1:item1"},
		{func() string { return CacheKeyItems("board1", "sticky_note", "") }, "items:board1:sticky_note:"},
		{func() string { return CacheKeyTags("board1") }, "tags:board1"},
		{func() string { return CacheKeyConnectors("board1") }, "connectors:board1"},
		{func() string { return CacheKeyUserInfo() }, "token:userinfo"},
	}

	for _, tt := range tests {
		got := tt.fn()
		if got != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, got)
		}
	}
}
