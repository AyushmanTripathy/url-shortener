package main

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

type Cache struct {
	cache *expirable.LRU[string, string]
}

func (c *Cache) init(entries int) {
	c.cache = expirable.NewLRU[string, string](entries, nil, time.Minute*10)
}

func (c *Cache) get(key string) (string, bool) {
	cachedUrl, ok := c.cache.Get(key)
	fmt.Println(cachedUrl, ok, "cache")
	if len(cachedUrl) < 2 {
		return cachedUrl, false
	}
	return cachedUrl, ok
}

func (c *Cache) set(key string, val string) {
	log.Println("Set", key, val)
	c.cache.Add(key, val)
}

func (c *Cache) invalidate(key string) {
	log.Println("Invalidated", key)
	c.cache.Remove(key)
}
