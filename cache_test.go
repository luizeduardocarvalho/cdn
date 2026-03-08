// cache_test.go
package main

import (
	"net/http"
	"testing"
	"time"
)

func TestCacheMissOnFirstGet(t *testing.T) {
	cache := NewCache()

	_, hit := cache.Get("/index.html")
	if hit {
		t.Error("expected cache miss on empty cache")
	}
}

func TestCacheHitAfterSet(t *testing.T) {
	cache := NewCache()

	cache.Set("/index.html", &CacheEntry{
		StatusCode: 200,
		Headers:    http.Header{"Content-Type": {"text/html"}},
		Body:       []byte("hello"),
		CreatedAt:  time.Now(),
		TTL:        10 * time.Second,
	})

	entry, hit := cache.Get("/index.html")
	if !hit {
		t.Fatal("expected cache hit after set")
	}
	if string(entry.Body) != "hello" {
		t.Errorf("expected body 'hello', got '%s'", string(entry.Body))
	}
	if entry.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", entry.StatusCode)
	}
}

func TestCacheMissAfterTTLExpires(t *testing.T) {
	cache := NewCache()

	cache.Set("/expire-me", &CacheEntry{
		StatusCode: 200,
		Headers:    http.Header{},
		Body:       []byte("temporary"),
		CreatedAt:  time.Now(),
		TTL:        50 * time.Millisecond,
	})

	// Should hit immediately
	_, hit := cache.Get("/expire-me")
	if !hit {
		t.Error("expected cache hit before TTL")
	}

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)

	_, hit = cache.Get("/expire-me")
	if hit {
		t.Error("expected cache miss after TTL expired")
	}
}

func TestCacheDifferentPaths(t *testing.T) {
	cache := NewCache()

	cache.Set("/page-a", &CacheEntry{
		StatusCode: 200,
		Headers:    http.Header{},
		Body:       []byte("page a"),
		CreatedAt:  time.Now(),
		TTL:        10 * time.Second,
	})

	_, hit := cache.Get("/page-a")
	if !hit {
		t.Error("expected hit for /page-a")
	}

	_, hit = cache.Get("/page-b")
	if hit {
		t.Error("expected miss for /page-b")
	}
}
