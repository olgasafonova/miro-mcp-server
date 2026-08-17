package miro

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// benchSerial resets the timer and runs fn b.N times.
func benchSerial(b *testing.B, fn func()) {
	b.Helper()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fn()
	}
}

// benchParallel resets the timer and runs fn from parallel goroutines.
func benchParallel(b *testing.B, fn func()) {
	b.Helper()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			fn()
		}
	})
}

// =============================================================================
// Cache Benchmarks
// =============================================================================

// newPopulatedBenchCache returns a cache holding the "test-key" entry.
func newPopulatedBenchCache() *Cache {
	cache := NewCache(DefaultCacheConfig())
	cache.Set("test-key", "test-value", time.Minute)
	return cache
}

func BenchmarkCache_Get(b *testing.B) {
	cache := newPopulatedBenchCache()
	benchSerial(b, func() { cache.Get("test-key") })
}

func BenchmarkCache_Set(b *testing.B) {
	cache := NewCache(DefaultCacheConfig())
	benchSerial(b, func() { cache.Set("test-key", "test-value", time.Minute) })
}

func BenchmarkCache_GetMiss(b *testing.B) {
	cache := NewCache(DefaultCacheConfig())
	benchSerial(b, func() { cache.Get("nonexistent-key") })
}

func BenchmarkCache_ParallelGet(b *testing.B) {
	cache := newPopulatedBenchCache()
	benchParallel(b, func() { cache.Get("test-key") })
}

func BenchmarkCache_ParallelSetGet(b *testing.B) {
	cache := NewCache(DefaultCacheConfig())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				cache.Set("key", "value", time.Minute)
			} else {
				cache.Get("key")
			}
			i++
		}
	})
}

func BenchmarkCache_InvalidatePrefix(b *testing.B) {
	cache := NewCache(DefaultCacheConfig())

	// Pre-populate with keys that match the prefix
	for i := 0; i < 100; i++ {
		cache.Set("items:board1:type"+string(rune('0'+i%10)), "value", time.Minute)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.InvalidatePrefix("items:board1")
		// Re-populate for next iteration
		for j := 0; j < 100; j++ {
			cache.Set("items:board1:type"+string(rune('0'+j%10)), "value", time.Minute)
		}
	}
}

// =============================================================================
// Circuit Breaker Benchmarks
// =============================================================================

func BenchmarkCircuitBreaker_Allow(b *testing.B) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	benchSerial(b, func() {
		cb.Allow()
		cb.RecordSuccess()
	})
}

func BenchmarkCircuitBreaker_ParallelAllow(b *testing.B) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	benchParallel(b, func() {
		if err := cb.Allow(); err == nil {
			cb.RecordSuccess()
		}
	})
}

func BenchmarkCircuitBreakerRegistry_Get(b *testing.B) {
	registry := NewCircuitBreakerRegistry(DefaultCircuitBreakerConfig())
	// Pre-create some breakers
	registry.Get("/boards")
	registry.Get("/items")
	registry.Get("/tags")

	benchSerial(b, func() { registry.Get("/boards") })
}

func BenchmarkCircuitBreakerRegistry_ParallelGet(b *testing.B) {
	registry := NewCircuitBreakerRegistry(DefaultCircuitBreakerConfig())
	endpoints := []string{"/boards", "/items", "/tags", "/connectors", "/frames"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			registry.Get(endpoints[i%len(endpoints)])
			i++
		}
	})
}

// =============================================================================
// Rate Limiter Benchmarks
// =============================================================================

func BenchmarkRateLimiter_Wait_NoDelay(b *testing.B) {
	rl := NewAdaptiveRateLimiter(DefaultRateLimiterConfig())
	ctx := context.Background()
	benchSerial(b, func() { rl.Wait(ctx) })
}

func BenchmarkRateLimiter_UpdateFromResponse(b *testing.B) {
	rl := NewAdaptiveRateLimiter(DefaultRateLimiterConfig())
	resp := &http.Response{Header: make(http.Header)}
	resp.Header.Set("X-RateLimit-Limit", "100")
	resp.Header.Set("X-RateLimit-Remaining", "50")

	benchSerial(b, func() { rl.UpdateFromResponse(resp) })
}

func BenchmarkRateLimiter_ParallelWait(b *testing.B) {
	rl := NewAdaptiveRateLimiter(DefaultRateLimiterConfig())
	ctx := context.Background()
	benchParallel(b, func() { rl.Wait(ctx) })
}

// =============================================================================
// Endpoint Extraction Benchmarks
// =============================================================================

func BenchmarkExtractEndpoint(b *testing.B) {
	paths := []string{
		"/boards/abc123/items/xyz789",
		"/boards/uXjVOXQCe5c=/sticky_notes",
		"/boards",
		"/boards/abc123/connectors/def456",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractEndpoint(paths[i%len(paths)])
	}
}

// =============================================================================
// Cache Key Benchmarks
// =============================================================================

func BenchmarkCacheKeyItem(b *testing.B) {
	benchSerial(b, func() { _ = CacheKeyItem("board123", "item456") })
}

func BenchmarkCacheKeyItems(b *testing.B) {
	benchSerial(b, func() { _ = CacheKeyItems("board123", "sticky_note", "cursor") })
}

// =============================================================================
// Combined Performance Tests
// =============================================================================

// benchPathEnv bundles the infrastructure a typical request path touches.
type benchPathEnv struct {
	cache      *Cache
	cbRegistry *CircuitBreakerRegistry
	rl         *AdaptiveRateLimiter
	ctx        context.Context
}

func newBenchPathEnv() *benchPathEnv {
	return &benchPathEnv{
		cache:      NewCache(DefaultCacheConfig()),
		cbRegistry: NewCircuitBreakerRegistry(DefaultCircuitBreakerConfig()),
		rl:         NewAdaptiveRateLimiter(DefaultRateLimiterConfig()),
		ctx:        context.Background(),
	}
}

// allowAndWait checks the circuit breaker and rate limiter for the item
// endpoint, returning the breaker so the caller can record the outcome.
func (e *benchPathEnv) allowAndWait(b *testing.B) *CircuitBreaker {
	cb := e.cbRegistry.Get("/boards/{id}/items/{id}")
	if err := cb.Allow(); err != nil {
		b.Fatal(err)
	}
	e.rl.Wait(e.ctx)
	return cb
}

// BenchmarkTypicalReadPath simulates a typical read operation with caching
func BenchmarkTypicalReadPath(b *testing.B) {
	env := newBenchPathEnv()

	// Pre-populate cache
	env.cache.Set("item:board1:item1", "cached-data", time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb := env.allowAndWait(b)

		// Check cache
		if _, ok := env.cache.Get("item:board1:item1"); !ok {
			b.Fatal("expected cache hit")
		}

		cb.RecordSuccess()
	}
}

// BenchmarkTypicalWritePath simulates a typical write operation with cache invalidation
func BenchmarkTypicalWritePath(b *testing.B) {
	env := newBenchPathEnv()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Pre-populate cache
		env.cache.Set("item:board1:item1", "cached-data", time.Minute)
		env.cache.Set("items:board1:sticky_note:", "list-data", time.Minute)

		cb := env.allowAndWait(b)

		// Simulate write - invalidate cache
		env.cache.InvalidateItem("board1", "item1")

		cb.RecordSuccess()
	}
}
