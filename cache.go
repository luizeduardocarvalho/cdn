package main

import (
	"net/http"
	"sync"
	"time"
)

type CacheEntry struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	CreatedAt  time.Time
	TTL        time.Duration
}

type Cache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
}

func NewCache() *Cache {
	return &Cache{
		entries: make(map[string]*CacheEntry),
	}
}

func (c *Cache) Get(key string) (*CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	if time.Since(entry.CreatedAt) > entry.TTL {
		return nil, false
	}

	return entry, true
}

func (c *Cache) Set(key string, entry *CacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = entry
}
