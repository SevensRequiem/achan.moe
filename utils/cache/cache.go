package cache

import "sync"

// CacheInterface defines the methods required for a cache
type CacheInterface interface {
	Set(key, value string)
	Get(key string) (string, bool)
	Delete(key string)
	ListAll() map[string]string
}

// Cache is a simple in-memory cache
type Cache struct {
	items map[string]string
	mu    sync.RWMutex
}

// New creates a new instance of Cache
func New() *Cache {
	return &Cache{
		items: make(map[string]string),
	}
}

// Set adds a value to the cache
func (c *Cache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = value
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, found := c.items[key]
	return value, found
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// ListAll returns all cache items
func (c *Cache) ListAll() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	itemsCopy := make(map[string]string)
	for k, v := range c.items {
		itemsCopy[k] = v
	}
	return itemsCopy
}
