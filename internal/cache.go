package internal

import (
	"log"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

type Cache struct {
	cache *expirable.LRU[string, string]
}

func (c *Cache) Init(entries int) {
	c.cache = expirable.NewLRU[string, string](entries, nil, time.Hour * 24)
}

func (c *Cache) Get(key string) (string, bool, bool) {
	cachedUrl, ok := c.cache.Get(key)
	return cachedUrl, ok, len(cachedUrl) >= 2
}

func (c *Cache) Set(key string, val string) {
	log.Println("Set", key, val)
	c.cache.Add(key, val)
}

func (c *Cache) Invalidate(key string) {
	log.Println("Invalidated", key)
	c.cache.Remove(key)
}
